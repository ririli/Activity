package redis

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/domain"
	rdb "github.com/redis/go-redis/v9"
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
	cache := &Cache{Client: client, LoadBounds: func(context.Context, string) (Bounds, error) {
		return Bounds{100, 1000}, nil
	}}
	g, err := cache.Start(ctx, edition)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Retire(ctx, edition)
	draw := func(user, id string, cost, at int64) domain.Fact {
		return domain.Fact{User: user, Draw: id, Consumed: fmt.Sprint(cost), Occurred: fmt.Sprint(at)}
	}
	apply := func(f domain.Fact) {
		t.Helper()
		if err := cache.Apply(ctx, g, f); err != nil {
			t.Fatal(err)
		}
	}

	const user = "9007199254740993"
	later, earlier := draw(user, "2", 100, 202), draw(user, "1", 5, 201)
	apply(later)
	errs := make(chan error, 64)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- cache.Apply(ctx, g, later)
			errs <- cache.Apply(ctx, g, earlier)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	apply(domain.Fact{User: user, Created: "100"})
	apply(draw("9007199254740992", "3", 105, 202))
	apply(draw("3", "4", 105, 201))
	apply(draw("3", "2", 101, 202)) // Redis only checks draw ID; MySQL verifies the fact.
	for _, at := range []string{"150", "160", "140"} {
		apply(domain.Fact{User: "7", Created: at})
	}
	b, err := cache.Board(ctx, g, user)
	if err != nil || b.Me == nil || b.Me.Score != 105 || b.Me.Reached != 202 || b.Me.Rank != 3 ||
		len(b.Top) != 4 || b.Top[0].User != "3" || b.Top[0].Score != 105 ||
		b.Top[1].User != "9007199254740992" || b.Top[3].Score != 0 || b.Top[3].Reached != 140 {
		t.Fatalf("unexpected ranking: %+v / %v", b, err)
	}
	if client.SCard(ctx, g.keys[3]).Val() != 4 {
		t.Fatal("draw Set should contain exactly four accepted draws")
	}
	for _, invalid := range []domain.Fact{draw("7", "5", 1<<53-1, 200), draw("7", "6", 1, 99)} {
		if err := cache.Apply(ctx, g, invalid); err == nil || client.SIsMember(ctx, g.keys[3], invalid.Draw).Val() {
			t.Fatalf("invalid fact touched the Set: %+v / %v", invalid, err)
		}
	}
	if err := client.HSet(ctx, g.keys[0], "run_id", "old-run").Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Board(ctx, g, user); err == nil {
		t.Fatal("accepted a generation from a different Redis process")
	}
	if err := cache.Apply(ctx, g, draw(user, "7", 1, 203)); err == nil {
		t.Fatal("wrote a generation from a different Redis process")
	}
}
