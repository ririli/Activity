package redis

import (
	"activity/pkg/ratelimit"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"net/netip"
	"strings"
	"time"
)

type Limiter struct {
	R         *redis.Client
	Whitelist []netip.Prefix
}

func NewLimiter(r *redis.Client, whitelist string) Limiter {
	l := Limiter{R: r}
	for _, raw := range strings.Split(whitelist, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		p, err := netip.ParsePrefix(raw)
		if err != nil {
			a, e := netip.ParseAddr(raw)
			if e != nil {
				panic(fmt.Errorf("REGISTER_IP_WHITELIST: %w", e))
			}
			a = a.Unmap()
			p = netip.PrefixFrom(a, a.BitLen())
		}
		l.Whitelist = append(l.Whitelist, p.Masked())
	}
	return l
}

// Preserve the deployed namespace: active counters survive this update.
var registerScript = redis.NewScript(`
local n = tonumber(redis.call('GET', KEYS[1]) or '0')
if n >= 3 then return {0, redis.call('PTTL', KEYS[1])} end
redis.call('INCR', KEYS[1])
redis.call('PEXPIRE', KEYS[1], 60000)
return {1, 0}
`)
var failuresScript = redis.NewScript(`
return {tonumber(redis.call('GET', KEYS[1]) or '0'), redis.call('PTTL', KEYS[1])}
`)
var recordFailureScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
redis.call('PEXPIRE', KEYS[1], 900000)
return n
`)

func (l Limiter) Register(ctx context.Context, ip string) error {
	a, err := netip.ParseAddr(ip)
	if err != nil {
		return fmt.Errorf("invalid registration source IP: %w", err)
	}
	a = a.Unmap()
	for _, p := range l.Whitelist {
		if p.Contains(a) {
			return nil
		}
	}
	ctx, cancel := context.WithTimeout(ctx, ratelimit.Budget)
	defer cancel()
	v, err := registerScript.Run(ctx, l.R, []string{"identity:register:" + a.String()}).Int64Slice()
	if err != nil {
		return err
	}
	if v[0] == 0 {
		return &ratelimit.Rejection{Status: 429, Code: "RATE_LIMITED", After: wait(v[1])}
	}
	return nil
}
func wait(ms int64) time.Duration {
	if ms < 1 {
		ms = 1
	}
	return time.Duration(ms) * time.Millisecond
}
func (l Limiter) LoginFailures(ctx context.Context, account string) (int, time.Duration, error) {
	ctx, cancel := context.WithTimeout(ctx, ratelimit.Budget)
	defer cancel()
	v, err := failuresScript.Run(ctx, l.R, []string{"identity:login:" + account}).Int64Slice()
	if err != nil {
		return 0, 0, err
	}
	return int(v[0]), wait(v[1]), nil
}
func (l Limiter) RecordLoginFailure(ctx context.Context, account string) error {
	ctx, cancel := context.WithTimeout(ctx, ratelimit.Budget)
	defer cancel()
	return recordFailureScript.Run(ctx, l.R, []string{"identity:login:" + account}).Err()
}
func (l Limiter) ClearLoginFailures(ctx context.Context, account string) error {
	ctx, cancel := context.WithTimeout(ctx, ratelimit.Budget)
	defer cancel()
	return l.R.Del(ctx, "identity:login:"+account).Err()
}
