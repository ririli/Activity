package identitygrpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"activity/services/ranking-service/internal/ranking/application"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func testClient(t *testing.T, now *time.Time) *Client {
	t.Helper()
	c, err := New("localhost:1", "token", time.Second, time.Minute, time.Hour, 100)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	c.now = func() time.Time { return *now }
	return c
}

func TestBatchGetCachesAndDeduplicates(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	c := testClient(t, &now)
	calls := 0
	var fetched []string
	c.fetch = func(_ context.Context, ids []string) (map[string]string, error) {
		calls++
		fetched = append([]string(nil), ids...)
		return map[string]string{"1": "alice", "2": "bob"}, nil
	}
	got, err := c.BatchGet(context.Background(), []string{"1", "2", "1"})
	if err != nil {
		t.Fatal(err)
	}
	if got["1"] != "alice" || got["2"] != "bob" {
		t.Fatalf("unexpected names %v", got)
	}
	if calls != 1 || len(fetched) != 2 {
		t.Fatalf("calls=%d fetched=%v", calls, fetched)
	}
	now = now.Add(30 * time.Second)
	if _, err := c.BatchGet(context.Background(), []string{"1", "2"}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected fresh cache hit, calls=%d", calls)
	}
	now = now.Add(31 * time.Second)
	if _, err := c.BatchGet(context.Background(), []string{"1"}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("expected refresh after ttl, calls=%d", calls)
	}
}

func TestBatchGetUsesStaleOnTransientError(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	c := testClient(t, &now)
	fail := false
	c.fetch = func(_ context.Context, _ []string) (map[string]string, error) {
		if fail {
			return nil, errors.New("identity down")
		}
		return map[string]string{"1": "alice"}, nil
	}
	if _, err := c.BatchGet(context.Background(), []string{"1"}); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	fail = true
	got, err := c.BatchGet(context.Background(), []string{"1"})
	if err == nil {
		t.Fatal("expected transient error to be reported")
	}
	if got["1"] != "alice" {
		t.Fatalf("expected stale name, got %v", got)
	}
}

func TestBatchGetPropagatesAuthFailure(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	c := testClient(t, &now)
	c.fetch = func(context.Context, []string) (map[string]string, error) {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}
	if _, err := c.BatchGet(context.Background(), []string{"1"}); !errors.Is(err, application.ErrProfileAuth) {
		t.Fatalf("expected ErrProfileAuth, got %v", err)
	}
}

func TestBatchGetCachesUnknownUserAsEmpty(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	c := testClient(t, &now)
	calls := 0
	c.fetch = func(context.Context, []string) (map[string]string, error) {
		calls++
		return map[string]string{}, nil
	}
	got, err := c.BatchGet(context.Background(), []string{"404"})
	if err != nil {
		t.Fatal(err)
	}
	if name, ok := got["404"]; !ok || name != "" {
		t.Fatalf("unknown user = %q, %v", name, ok)
	}
	if _, err := c.BatchGet(context.Background(), []string{"404"}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("unknown user should be negatively cached, calls=%d", calls)
	}
}
