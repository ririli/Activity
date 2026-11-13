package ratelimit

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// One atomic operation, with the Redis clock shared by every application instance.
var bucket = redis.NewScript(`
local clock = redis.call('TIME')
local now = tonumber(clock[1]) * 1000 + math.floor(tonumber(clock[2]) / 1000)
local rate, capacity = tonumber(ARGV[1]), tonumber(ARGV[2])
local state = redis.call('HMGET', KEYS[1], 'tokens', 'at')
local tokens, at = tonumber(state[1]), tonumber(state[2])
if not tokens or not at then tokens, at = capacity, now end
now = math.max(now, at)
tokens = math.min(capacity, tokens + (now - at) * rate / 1000)
local allowed, retry = 0, 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
else
  retry = math.ceil((1 - tokens) * 1000 / rate)
end
redis.call('HSET', KEYS[1], 'tokens', tostring(tokens), 'at', now)
redis.call('PEXPIRE', KEYS[1], math.ceil(2 * capacity * 1000 / rate))
return {allowed, retry}
`)

type Redis struct{ Client *redis.Client }

// NewRedis owns a separate pool, so a slow limiter cannot exhaust business pools.
func NewRedis() *Redis {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:16379"
	}
	return &Redis{Client: redis.NewClient(&redis.Options{
		Addr: addr, Password: os.Getenv("REDIS_PASSWORD"), MaxRetries: -1,
		DialTimeout: Budget, ReadTimeout: Budget, WriteTimeout: Budget,
		PoolTimeout: Budget, PoolSize: 16, ContextTimeoutEnabled: true,
	})}
}

func (r *Redis) Close() error { return r.Client.Close() }

func (r *Redis) Allow(ctx context.Context, key string, rule Rule) (Result, error) {
	if err := rule.Validate(); err != nil {
		return Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, Budget)
	defer cancel()
	v, err := bucket.Run(ctx, r.Client, []string{key}, rule.Rate, rule.Capacity).Int64Slice()
	if err != nil {
		return Result{}, err
	}
	if len(v) != 2 {
		return Result{}, fmt.Errorf("unexpected rate limit result")
	}
	return Result{Allowed: v[0] == 1, RetryAfter: time.Duration(v[1]) * time.Millisecond}, nil
}
