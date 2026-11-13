package redis

import (
	"activity/pkg/hourly"
	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/domain"
	"context"
	"fmt"
	"os"
	"testing"

	rdb "github.com/redis/go-redis/v9"
)

func TestHourlyWindowAndShardedBoard(t *testing.T) {
	addr := os.Getenv("RANKING_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("requires disposable Redis")
	}
	ctx := context.Background()
	client := rdb.NewClient(&rdb.Options{Addr: addr})
	defer client.Close()
	var clients [hourly.Shards]*rdb.Client
	for i := range clients {
		clients[i] = client
	}
	load := func(context.Context, string) (Bounds, error) { return Bounds{Min: 100, End: 1000}, nil }
	edition := fmt.Sprint(platform.ID())
	cache := HourlyCache{Clients: clients, LoadBounds: load, Namespace: "hourly-integration"}
	old := hourly.Window{Edition: edition, Cutoff: 400}
	next := hourly.Window{Edition: edition, Cutoff: 800}
	defer cache.Retire(ctx, old, 1)
	defer cache.Retire(ctx, next, 1)
	fact := func(window hourly.Window, seq int64, score, at string) domain.Fact {
		return domain.Fact{Edition: edition, User: "1", Draw: "100", Total: score, Reached: at, Hourly: &hourly.Position{Cutoff: window.Cutoff, Shard: 1, Lane: 0, Seq: seq}}
	}
	first, second := fact(old, 1, "5", "200"), fact(old, 2, "20", "399")
	if last, err := cache.Apply(ctx, old, 1, 0, []domain.Fact{second}); err != nil || last != 0 {
		t.Fatalf("out-of-order advanced: %d %v", last, err)
	}
	if _, err := cache.Apply(ctx, next, 1, 0, []domain.Fact{fact(next, 1, "25", "400")}); err != nil {
		t.Fatal(err)
	}
	var mark [hourly.Lanes]int64
	mark[0] = 2
	if _, err := cache.Freeze(ctx, old, 1, mark); err == nil {
		t.Fatal("froze with a gap")
	}
	if last, err := cache.Apply(ctx, old, 1, 0, []domain.Fact{first, second, second}); err != nil || last != 2 {
		t.Fatalf("replay: %d %v", last, err)
	}
	entries, err := cache.Freeze(ctx, old, 1, mark)
	if err != nil || len(entries) != 1 || entries[0].Score != 20 || entries[0].Reached != 399 {
		t.Fatalf("old result: %+v %v", entries, err)
	}
	key := fmt.Sprintf("ranking:%s:{%s:1}:hourly:%d:board", cache.Namespace, edition, old.Cutoff)
	if err := client.Del(ctx, key).Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Freeze(ctx, old, 1, mark); err == nil {
		t.Fatal("froze after losing its ZSET")
	}
	if _, err := cache.Apply(ctx, old, 1, 0, []domain.Fact{first, second}); err != nil {
		t.Fatal(err)
	}
	if rebuilt, err := cache.Freeze(ctx, old, 1, mark); err != nil || len(rebuilt) != 1 || rebuilt[0] != entries[0] {
		t.Fatalf("rebuild: %+v %v", rebuilt, err)
	}
	live := Cache{Client: client, Clients: clients, LoadBounds: load}
	generation, err := live.Start(ctx, edition)
	if err != nil {
		t.Fatal(err)
	}
	defer live.Retire(ctx, edition)
	for id := 1; id <= 104; id++ {
		if err := live.Apply(ctx, generation, domain.Fact{User: fmt.Sprint(id), Created: "200"}); err != nil {
			t.Fatal(err)
		}
	}
	for _, user := range []int{1, 20, 99, 100, 101, 104} {
		board, err := live.Board(ctx, generation, fmt.Sprint(user))
		if err != nil || board.Me == nil || board.Me.Rank != int64(user) || len(board.Top) != 20 || board.Top[19].User != "20" {
			t.Fatalf("sharded board for %d: %+v %v", user, board, err)
		}
	}
}
