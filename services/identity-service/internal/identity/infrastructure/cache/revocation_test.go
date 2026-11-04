package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeKV struct {
	exists     bool
	err        error
	calls      int
	setCalls   int
	lastKey    string
	lastTTL    time.Duration
	setCallsFn func()
}

func (f *fakeKV) Exists(context.Context, string) (bool, error) {
	f.calls++
	return f.exists, f.err
}

func (f *fakeKV) Set(_ context.Context, key string, ttl time.Duration) error {
	f.setCalls++
	f.lastKey = key
	f.lastTTL = ttl
	if f.setCallsFn != nil {
		f.setCallsFn()
	}
	return nil
}

func TestNegativeCacheSkipsRedisButExpires(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	kv := &fakeKV{}
	c, err := newRevocationCache(kv, Config{NegativeTTL: 2 * time.Second, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	expiry := now.Add(time.Minute)
	if revoked, err := c.Revoked(context.Background(), "sid-1", expiry); err != nil || revoked {
		t.Fatalf("first check = %v, %v", revoked, err)
	}
	if kv.calls != 1 {
		t.Fatalf("expected one redis call, got %d", kv.calls)
	}
	if revoked, err := c.Revoked(context.Background(), "sid-1", expiry); err != nil || revoked {
		t.Fatalf("cached check = %v, %v", revoked, err)
	}
	if kv.calls != 1 {
		t.Fatalf("negative cache did not skip redis: %d", kv.calls)
	}
	now = now.Add(3 * time.Second)
	if revoked, err := c.Revoked(context.Background(), "sid-1", expiry); err != nil || revoked {
		t.Fatalf("expired check = %v, %v", revoked, err)
	}
	if kv.calls != 2 {
		t.Fatalf("expected redis after negative ttl, got %d", kv.calls)
	}
}

func TestRevokedRedisKeyWinsAndClearsNegative(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	kv := &fakeKV{}
	c, err := newRevocationCache(kv, Config{NegativeTTL: 2 * time.Second, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	expiry := now.Add(time.Minute)
	if _, err := c.Revoked(context.Background(), "sid-2", expiry); err != nil {
		t.Fatal(err)
	}
	// A logout on another instance writes Redis; this instance's short negative
	// entry is bypassed only when it expires or is explicitly forgotten.
	c.Forget("sid-2")
	kv.exists = true
	if revoked, err := c.Revoked(context.Background(), "sid-2", expiry); err != nil || !revoked {
		t.Fatalf("expected revoked, got %v, %v", revoked, err)
	}
	// A revoked result deletes the negative entry, so another check consults Redis.
	if revoked, err := c.Revoked(context.Background(), "sid-2", expiry); err != nil || !revoked {
		t.Fatalf("expected revoked again, got %v, %v", revoked, err)
	}
	if kv.calls != 3 {
		t.Fatalf("expected redis on each revoked check, got %d", kv.calls)
	}
}

func TestRedisErrorIsNotTreatedAsNotRevoked(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	kv := &fakeKV{err: errors.New("redis down")}
	c, err := newRevocationCache(kv, Config{NegativeTTL: 2 * time.Second, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Revoked(context.Background(), "sid-3", now.Add(time.Minute)); err == nil {
		t.Fatal("expected redis error")
	}
	if _, err := c.Revoked(context.Background(), "sid-3", now.Add(time.Minute)); err == nil {
		t.Fatal("expected redis error again; errors must not be negatively cached")
	}
	if kv.calls != 2 {
		t.Fatalf("expected redis retry, got %d", kv.calls)
	}
}

func TestTokenExpiryCapsNegativeEntry(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	kv := &fakeKV{}
	c, err := newRevocationCache(kv, Config{NegativeTTL: 2 * time.Second, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Revoked(context.Background(), "sid-4", now.Add(500*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Revoked(context.Background(), "sid-4", now.Add(500*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if kv.calls != 1 {
		t.Fatalf("expected token-expiry negative cache hit, got %d", kv.calls)
	}
	now = now.Add(time.Second)
	if _, err := c.Revoked(context.Background(), "sid-4", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if kv.calls != 2 {
		t.Fatalf("expected re-check after token expiry, got %d", kv.calls)
	}
}

func TestMarkRevokedUsesSessionExpiryAndPadding(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	kv := &fakeKV{}
	c, err := newRevocationCache(kv, Config{NegativeTTL: 2 * time.Second, Padding: 30 * time.Second, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	expires := now.Add(10 * time.Second)
	if err := c.MarkRevoked(context.Background(), "sid-5", expires); err != nil {
		t.Fatal(err)
	}
	if kv.setCalls != 1 {
		t.Fatalf("expected one set, got %d", kv.setCalls)
	}
	if kv.lastKey != "identity:revoke:sid:sid-5" {
		t.Fatalf("unexpected key %q", kv.lastKey)
	}
	if kv.lastTTL != 40*time.Second {
		t.Fatalf("unexpected ttl %v", kv.lastTTL)
	}
}
