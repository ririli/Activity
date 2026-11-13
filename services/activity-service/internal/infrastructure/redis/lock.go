package redis

import (
	"activity/pkg/platform"
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

type Redis struct{ Client *redis.Client }

var releaseScript = `if redis.call('GET',KEYS[1])==ARGV[1] then return redis.call('DEL',KEYS[1]) end return 0`

// Acquire takes the per-user mutation lock used by activity commands. It never
// waits; a held lock maps to OPERATION_IN_PROGRESS at the caller.
func (l Redis) Acquire(ctx context.Context, user string) (func(), error) {
	key := "activity:" + platform.Env("APP_ENV", "local") + ":user:" + user + ":mutation"
	token := platform.Random()
	ok, err := l.Client.SetNX(ctx, key, token, 10*time.Second).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, platform.Fail(409, "OPERATION_IN_PROGRESS")
	}
	return func() {
		release, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		l.Client.Eval(release, releaseScript, []string{key}, token)
	}, nil
}
