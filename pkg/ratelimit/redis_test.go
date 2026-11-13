package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type brokenLimiter struct{}

func (brokenLimiter) Allow(context.Context, string, Rule) (Result, error) {
	return Result{}, errors.New("offline")
}

func TestFailurePolicies(t *testing.T) {
	t.Setenv("RATE_LIMIT_MODE", "enforce")
	for _, open := range []bool{false, true} {
		g := Gate{Limiter: brokenLimiter{}, Service: "test", Group: "read", Rule: Rule{1, 1}, FailOpen: open}
		err := g.Check(context.Background(), "42")
		if open && err != nil {
			t.Fatal(err)
		}
		if !open {
			var e *Rejection
			if !errors.As(err, &e) || e.Status != 503 {
				t.Fatalf("%v", err)
			}
		}
	}
}

func TestObserveMode(t *testing.T) {
	t.Setenv("RATE_LIMIT_MODE", "observe")
	g := Gate{Limiter: brokenLimiter{}, Service: "test", Group: "observe", Rule: Rule{1, 1}}
	if err := g.Check(context.Background(), "42"); err != nil {
		t.Fatal(err)
	}
	g.Limiter = deniedLimiter{}
	if err := g.Check(context.Background(), "42"); err != nil {
		t.Fatal(err)
	}
}

type deniedLimiter struct{}

func (deniedLimiter) Allow(context.Context, string, Rule) (Result, error) {
	return Result{Allowed: false, RetryAfter: time.Second}, nil
}

func TestSlowRedisIsBounded(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() { defer conn.Close(); <-done }()
		}
	}()
	t.Setenv("REDIS_ADDR", listener.Addr().String())
	r := NewRedis()
	defer r.Close()
	start := time.Now()
	_, err = r.Allow(context.Background(), "test:timeout", Rule{1, 1})
	if err == nil {
		t.Fatal("silent dependency succeeded")
	}
	if elapsed := time.Since(start); elapsed > 350*time.Millisecond {
		t.Fatalf("request exceeded budget with scheduling allowance: %v", elapsed)
	}
}

func TestRedisAtomicBucket(t *testing.T) {
	addr := os.Getenv("RATE_LIMIT_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set RATE_LIMIT_TEST_REDIS_ADDR for Redis integration")
	}
	ctx := context.Background()
	a := &Redis{Client: redis.NewClient(&redis.Options{Addr: addr, MaxRetries: -1})}
	b := &Redis{Client: redis.NewClient(&redis.Options{Addr: addr, MaxRetries: -1})}
	defer a.Close()
	defer b.Close()
	key := fmt.Sprintf("rl:test:atomic:%d", time.Now().UnixNano())
	defer a.Client.Del(ctx, key, key+":other")
	// An almost stationary bucket makes the admission count deterministic.
	rule := Rule{Rate: 0.01, Capacity: 4}
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := a
			if i%2 == 0 {
				r = b
			}
			v, err := r.Allow(ctx, key, rule)
			if err != nil {
				t.Error(err)
				return
			}
			if v.Allowed {
				allowed.Add(1)
			} else if v.RetryAfter <= 0 {
				t.Error("missing retry delay")
			}
		}(i)
	}
	wg.Wait()
	if allowed.Load() != 4 {
		t.Fatalf("admitted %d, expected 4", allowed.Load())
	}
	if ttl := a.Client.PTTL(ctx, key).Val(); ttl <= 0 || ttl > 800*time.Second {
		t.Fatalf("TTL %v", ttl)
	}
	if v, err := b.Allow(ctx, key+":other", rule); err != nil || !v.Allowed {
		t.Fatalf("key isolation: %+v %v", v, err)
	}
	// Simulate elapsed Redis time without sleeping; a rejected request must not
	// turn the refill timestamp into a cooldown deadline.
	now, err := a.Client.Time(ctx).Result()
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Client.HSet(ctx, key, "at", now.Add(-200*time.Second).UnixMilli()).Err(); err != nil {
		t.Fatal(err)
	}
	if v, err := b.Allow(ctx, key, rule); err != nil || !v.Allowed {
		t.Fatalf("refill: %+v %v", v, err)
	}
	if err := a.Client.PExpire(ctx, key, time.Millisecond).Err(); err != nil {
		t.Fatal(err)
	}
	// Redis lazy expiry is evaluated against server time on the next request.
	time.Sleep(5 * time.Millisecond)
	if v, err := a.Allow(ctx, key, rule); err != nil || !v.Allowed {
		t.Fatalf("expired bucket: %+v %v", v, err)
	}
}
