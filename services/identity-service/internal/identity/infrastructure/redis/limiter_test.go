package redis

import (
	"activity/pkg/ratelimit"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	rdb "github.com/redis/go-redis/v9"
)

func TestAccountCounters(t *testing.T) {
	addr := os.Getenv("RATE_LIMIT_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set RATE_LIMIT_TEST_REDIS_ADDR for Redis integration")
	}
	ctx := context.Background()
	r := rdb.NewClient(&rdb.Options{Addr: addr})
	defer r.Close()
	l := NewLimiter(r, "192.0.2.0/24,2001:db8::1")
	// Unique IPv6 source also tests canonical IP normalization.
	ip := fmt.Sprintf("2001:db8:1::%x", time.Now().UnixNano()&65535)
	key := "identity:register:" + ip
	account := fmt.Sprintf("test-%d", time.Now().UnixNano())
	defer r.Del(ctx, key, "identity:login:"+account)
	for i := 0; i < 3; i++ {
		if err := l.Register(ctx, ip); err != nil {
			t.Fatal(err)
		}
	}
	// Reduce the TTL to distinguish no-op rejection from a 60-second renewal.
	r.PExpire(ctx, key, 40*time.Second)
	for i := 0; i < 4; i++ {
		var rejection *ratelimit.Rejection
		if err := l.Register(ctx, ip); !errors.As(err, &rejection) || rejection.After <= 0 {
			t.Fatalf("%v", err)
		}
	}
	if n := r.Get(ctx, key).Val(); n != "3" {
		t.Fatalf("counter = %s", n)
	}
	if ttl := r.PTTL(ctx, key).Val(); ttl > 40*time.Second || ttl <= 0 {
		t.Fatalf("rejection renewed TTL: %v", ttl)
	}
	for i := 0; i < 5; i++ {
		if err := l.RecordLoginFailure(ctx, account); err != nil {
			t.Fatal(err)
		}
	}
	n, ttl, err := l.LoginFailures(ctx, account)
	if err != nil || n != 5 || ttl < 14*time.Minute {
		t.Fatalf("%d %v %v", n, ttl, err)
	}
	r.PExpire(ctx, "identity:login:"+account, time.Minute)
	if _, ttl, err = l.LoginFailures(ctx, account); err != nil || ttl > time.Minute {
		t.Fatalf("read renewed TTL: %v %v", ttl, err)
	}
	if err := l.ClearLoginFailures(ctx, account); err != nil {
		t.Fatal(err)
	}
	if n, _, err := l.LoginFailures(ctx, account); err != nil || n != 0 {
		t.Fatalf("clear: %d %v", n, err)
	}
	// A whitelist bypasses registration only, including with no Redis connection.
	white := NewLimiter(nil, "192.0.2.0/24,2001:db8::1")
	for _, ip := range []string{"192.0.2.3", "::ffff:192.0.2.3", "2001:db8::1"} {
		if err := white.Register(ctx, ip); err != nil {
			t.Fatal(err)
		}
	}
}
