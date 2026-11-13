// Package ratelimit provides transport-independent distributed admission control.
package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const Budget = 100 * time.Millisecond

type Rule struct {
	Rate     float64 // Tokens replenished per second.
	Capacity int
}

func (r Rule) Validate() error {
	if r.Rate <= 0 || math.IsNaN(r.Rate) || math.IsInf(r.Rate, 0) || r.Capacity < 1 || r.Capacity > 1000000 || r.Rate < 0.001 || r.Rate > 1000000 {
		return errors.New("invalid rate limit rule")
	}
	return nil
}

// Configure accepts <prefix>_RATE and <prefix>_CAPACITY. Invalid settings fail startup.
func Configure(prefix string, rule Rule) Rule {
	switch os.Getenv("RATE_LIMIT_MODE") {
	case "", "enforce", "observe":
	default:
		panic("RATE_LIMIT_MODE must be enforce or observe")
	}
	if raw := os.Getenv(prefix + "_RATE"); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			panic(fmt.Errorf("%s_RATE: %w", prefix, err))
		}
		rule.Rate = v
	}
	if raw := os.Getenv(prefix + "_CAPACITY"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			panic(fmt.Errorf("%s_CAPACITY: %w", prefix, err))
		}
		rule.Capacity = v
	}
	if err := rule.Validate(); err != nil {
		panic(fmt.Errorf("%s: %w", prefix, err))
	}
	return rule
}

type Result struct {
	Allowed    bool
	RetryAfter time.Duration
}

type Limiter interface {
	Allow(context.Context, string, Rule) (Result, error)
}

// Rejection can be mapped by any transport without a dependency on Gin.
type Rejection struct {
	Status int
	Code   string
	After  time.Duration
}

func (e *Rejection) Error() string             { return e.Code }
func (e *Rejection) HTTPStatus() int           { return e.Status }
func (e *Rejection) ErrorCode() string         { return e.Code }
func (e *Rejection) RetryAfter() time.Duration { return e.After }

type Gate struct {
	Limiter        Limiter
	Service, Group string
	Rule           Rule
	FailOpen       bool
}

type counters struct {
	requests, rejected, errors, micros atomic.Uint64
}

// Only service/group constants are labels; neither users nor raw URLs become
// telemetry dimensions. Quotas themselves always live in Redis.
var telemetry sync.Map

func (g Gate) record(elapsed time.Duration, result Result, err error) {
	v, _ := telemetry.LoadOrStore(g.Service+":"+g.Group, &counters{})
	c := v.(*counters)
	n := c.requests.Add(1)
	c.micros.Add(uint64(elapsed.Microseconds()))
	if err != nil {
		c.errors.Add(1)
	} else if !result.Allowed {
		c.rejected.Add(1)
	}
	if n%100 == 0 || err != nil || !result.Allowed {
		slog.Info("rate_limit_totals", "service", g.Service, "group", g.Group,
			"requests", n, "rejected", c.rejected.Load(), "errors", c.errors.Load(),
			"mean_duration_ms", float64(c.micros.Load())/float64(n)/1000)
	}
}

func (g Gate) Check(ctx context.Context, user string) error {
	// Subjects come exclusively from the authenticated principal.
	if user == "" {
		return &Rejection{Status: 401, Code: "INVALID_SESSION"}
	}
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}
	key := strings.Join([]string{"rl", env, g.Service, g.Group, user}, ":")
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, Budget)
	defer cancel()
	var result Result
	var err error
	if g.Limiter == nil {
		err = errors.New("rate limiter not configured")
	} else {
		result, err = g.Limiter.Allow(ctx, key, g.Rule)
	}
	g.record(time.Since(start), result, err)
	observe := os.Getenv("RATE_LIMIT_MODE") == "observe"
	attrs := []any{"service", g.Service, "group", g.Group, "duration_ms", float64(time.Since(start).Microseconds()) / 1000}
	if err != nil {
		slog.Warn("rate_limit_dependency_error", append(attrs, "fail_open", g.FailOpen, "observe", observe, "error", err)...)
		if g.FailOpen || observe {
			return nil
		}
		return &Rejection{Status: 503, Code: "DEPENDENCY_UNAVAILABLE"}
	}
	if !result.Allowed {
		slog.Warn("rate_limit_rejected", append(attrs, "observe", observe)...)
		if observe {
			return nil
		}
		return &Rejection{Status: 429, Code: "RATE_LIMITED", After: result.RetryAfter}
	}
	slog.Debug("rate_limit_allowed", attrs...)
	return nil
}
