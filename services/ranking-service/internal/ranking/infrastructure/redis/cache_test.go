package redis

import (
	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/domain"
	"context"
	"fmt"
	rdb "github.com/redis/go-redis/v9"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestAtomicRanking(t *testing.T) {
	addr := os.Getenv("RANKING_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set RANKING_TEST_REDIS_ADDR for real Lua integration tests")
	}
	ctx := context.Background()
	client := rdb.NewClient(&rdb.Options{Addr: addr})
	defer client.Close()
	edition := fmt.Sprint(platform.ID())
	cache := &Cache{Client: client, LoadBounds: func(context.Context, string) (Bounds, error) { return Bounds{100, 1000}, nil }}
	defer client.Del(ctx, cache.keys(edition)...)
	event := func(user, draw string, cost, at int) platform.Event {
		f := domain.Fact{User: user, Edition: edition, Draw: draw, Seq: draw, Consumed: fmt.Sprint(cost), Occurred: fmt.Sprint(at)}
		return platform.Event{Type: "FormalDrawCommitted.v1", Payload: platform.JSON(f)}
	}
	init := func(user string, at int) platform.Event {
		return platform.Event{Type: "ParticipationInitialized.v1", Payload: platform.JSON(domain.Fact{User: user, Edition: edition, Created: fmt.Sprint(at)})}
	}
	apply := func(ev platform.Event) {
		t.Helper()
		if e := cache.Apply(ctx, ev); e != nil {
			t.Fatal(e)
		}
	}
	// Later draw arrives first; both HTTP and MQ repeat every draw concurrently.
	later := event("9007199254740993", "2", 100, 202)
	earlier := event("9007199254740993", "1", 5, 201)
	errs := make(chan error, 64)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- cache.Apply(ctx, later); errs <- cache.Apply(ctx, earlier) }()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	apply(init("9007199254740993", 100))
	b, e := cache.Board(ctx, edition, "9007199254740993")
	if e != nil {
		t.Fatal(e)
	}
	if b.Me == nil || b.Me.Score != 105 || b.Me.Reached != 202 || b.Me.Rank != 1 {
		t.Fatalf("incorrect dedup/order: %+v", b.Me)
	}
	// Same score and second: smaller BIGINT ID comes first without Lua rounding.
	apply(event("9007199254740992", "3", 105, 202))
	apply(event("3", "4", 105, 201))
	b, e = cache.Board(ctx, edition, "9007199254740993")
	if e != nil {
		t.Fatal(e)
	}
	if len(b.Top) != 3 || b.Top[0].User != "3" || b.Top[1].User != "9007199254740992" || b.Me.Rank != 3 {
		t.Fatalf("invalid ties: %+v", b)
	}
	// Reusing a draw with another user, amount, or time must fail without changes.
	if e = cache.Apply(ctx, event("3", "2", 101, 202)); e == nil || !strings.Contains(e.Error(), "CONFLICT") {
		t.Fatalf("conflicting draw accepted: %v", e)
	}
	b, _ = cache.Board(ctx, edition, "3")
	if b.Me.Score != 105 {
		t.Fatal("conflict modified state")
	}
	// Zero-score initialization is idempotent and retains earliest creation time.
	apply(init("7", 150))
	apply(init("7", 160))
	apply(init("7", 140))
	b, _ = cache.Board(ctx, edition, "7")
	if b.Me.Score != 0 || b.Me.Reached != 140 {
		t.Fatalf("invalid initialization %+v", b.Me)
	}
	// Overflow disables cache and leaves SQL available; the draw is not deduped.
	apply(event("7", "5", 1<<53-1, 200))
	if _, e = cache.Board(ctx, edition, "7"); e == nil {
		t.Fatal("precision failure did not disable cache")
	}
	if n := client.HExists(ctx, cache.keys(edition)[2], "5").Val(); n {
		t.Fatal("overflow marked as applied")
	}
}
func TestInvalidTimeDoesNotWrite(t *testing.T) {
	addr := os.Getenv("RANKING_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("requires Redis")
	}
	ctx := context.Background()
	client := rdb.NewClient(&rdb.Options{Addr: addr})
	defer client.Close()
	cache := &Cache{Client: client, LoadBounds: func(context.Context, string) (Bounds, error) { return Bounds{100, 200}, nil }}
	edition := fmt.Sprint(platform.ID())
	defer client.Del(ctx, cache.keys(edition)...)
	e := cache.Apply(ctx, platform.Event{Type: "ParticipationInitialized.v1", Payload: platform.JSON(domain.Fact{User: "1", Edition: edition, Created: "99"})})
	if e == nil {
		t.Fatal("out of bounds accepted")
	}
	if client.Exists(ctx, cache.keys(edition)...).Val() != 0 {
		t.Fatal("invalid event wrote cache")
	}
}
