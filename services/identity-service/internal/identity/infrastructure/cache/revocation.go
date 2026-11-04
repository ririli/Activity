package cache

import (
	"context"
	"encoding/json"
	"time"

	"activity/services/identity-service/internal/identity/application"

	"github.com/allegro/bigcache/v3"
	"github.com/redis/go-redis/v9"
)

// Config controls the local negative cache and the Redis revocation TTL.
type Config struct {
	NegativeTTL time.Duration // default 2s
	LifeWindow  time.Duration // default 2s
	CleanWindow time.Duration // default 1s
	HardMaxMB   int           // default 16
	Padding     time.Duration // default 30s
	Now         func() time.Time
}

type redisKV interface {
	Exists(ctx context.Context, key string) (bool, error)
	Set(ctx context.Context, key string, ttl time.Duration) error
}

type redisAdapter struct{ client *redis.Client }

func (r redisAdapter) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	return n > 0, err
}

func (r redisAdapter) Set(ctx context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	return r.client.Set(ctx, key, "1", ttl).Err()
}

type negativeEntry struct {
	ExpiresAt int64 `json:"expires_at"`
	CachedAt  int64 `json:"cached_at"`
}

// RevocationCache implements application.RevocationStore. The Redis key is the
// authoritative fast rejection path; BigCache only remembers a recent Redis
// miss for a very short time and never replaces the MySQL session check.
type RevocationCache struct {
	kv       redisKV
	negative *bigcache.BigCache
	cfg      Config
}

func NewRevocationCache(client *redis.Client, cfg Config) (*RevocationCache, error) {
	return newRevocationCache(redisAdapter{client: client}, cfg)
}

func newRevocationCache(kv redisKV, cfg Config) (*RevocationCache, error) {
	if cfg.NegativeTTL <= 0 {
		cfg.NegativeTTL = 2 * time.Second
	}
	if cfg.LifeWindow <= 0 {
		cfg.LifeWindow = 2 * time.Second
	}
	if cfg.CleanWindow <= 0 {
		cfg.CleanWindow = time.Second
	}
	if cfg.HardMaxMB <= 0 {
		cfg.HardMaxMB = 16
	}
	if cfg.Padding <= 0 {
		cfg.Padding = 30 * time.Second
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	negative, err := bigcache.New(context.Background(), bigcache.Config{
		Shards:           256,
		LifeWindow:       cfg.LifeWindow,
		CleanWindow:      cfg.CleanWindow,
		HardMaxCacheSize: cfg.HardMaxMB,
		Verbose:          false,
	})
	if err != nil {
		return nil, err
	}
	return &RevocationCache{kv: kv, negative: negative, cfg: cfg}, nil
}

func revocationKey(sid string) string { return "identity:revoke:sid:" + sid }

func (c *RevocationCache) Revoked(ctx context.Context, sid string, tokenExpiry time.Time) (bool, error) {
	now := c.cfg.Now()
	if raw, err := c.negative.Get(sid); err == nil {
		var v negativeEntry
		if json.Unmarshal(raw, &v) == nil {
			cachedUntil := time.Unix(0, v.CachedAt).Add(c.cfg.NegativeTTL)
			expiresAt := time.Unix(0, v.ExpiresAt)
			if now.Before(cachedUntil) && now.Before(expiresAt) {
				return false, nil
			}
		}
	}
	revoked, err := c.kv.Exists(ctx, revocationKey(sid))
	if err != nil {
		return false, err
	}
	if revoked {
		_ = c.negative.Delete(sid)
		return true, nil
	}
	until := now.Add(c.cfg.NegativeTTL)
	if !tokenExpiry.IsZero() && tokenExpiry.Before(until) {
		until = tokenExpiry
	}
	if until.After(now) {
		raw, err := json.Marshal(negativeEntry{ExpiresAt: until.UnixNano(), CachedAt: now.UnixNano()})
		if err == nil {
			_ = c.negative.Set(sid, raw)
		}
	}
	return false, nil
}

func (c *RevocationCache) MarkRevoked(ctx context.Context, sid string, sessionExpiresAt time.Time) error {
	ttl := sessionExpiresAt.Add(c.cfg.Padding).Sub(c.cfg.Now())
	err := c.kv.Set(ctx, revocationKey(sid), ttl)
	c.Forget(sid)
	return err
}

func (c *RevocationCache) Forget(sid string) {
	_ = c.negative.Delete(sid)
}

var _ application.RevocationStore = (*RevocationCache)(nil)
