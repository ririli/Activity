package redis

import (
	"activity/pkg/platform"
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

type Limiter struct{ R *redis.Client }

func (l Limiter) Take(ctx context.Context, key string, max int, window time.Duration) error {
	v, e := l.R.Eval(ctx, `local n=redis.call('INCR',KEYS[1]);redis.call('PEXPIRE',KEYS[1],ARGV[1]);return n`, []string{"identity:" + key}, window.Milliseconds()).Int()
	if e != nil {
		return e
	}
	if v > max {
		return platform.Fail(429, "RATE_LIMITED")
	}
	return nil
}
func (l Limiter) Failures(ctx context.Context, key string) (int, error) {
	n, e := l.R.Get(ctx, "identity:login:"+key).Int()
	if e == redis.Nil {
		return 0, nil
	}
	return n, e
}
func (l Limiter) Clear(ctx context.Context, key string) error {
	return l.R.Del(ctx, "identity:login:"+key).Err()
}
