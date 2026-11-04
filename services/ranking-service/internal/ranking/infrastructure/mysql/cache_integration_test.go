package mysql

import (
	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/domain"
	cache "activity/services/ranking-service/internal/ranking/infrastructure/redis"
	"context"
	"database/sql"
	"fmt"
	driver "github.com/go-sql-driver/mysql"
	rdb "github.com/redis/go-redis/v9"
	"os"
	"testing"
)

func TestSQLAndRedisDelta(t *testing.T) {
	dsn := os.Getenv("RANKING_TEST_MYSQL_DSN")
	addr := os.Getenv("RANKING_TEST_REDIS_ADDR")
	if dsn == "" || addr == "" || os.Getenv("RANKING_TEST_ALLOW_SCHEMA") != "1" {
		t.Skip("requires empty, disposable MySQL schema and Redis")
	}
	ctx := context.Background()
	cfg, e := driver.ParseDSN(dsn)
	if e != nil {
		t.Fatal(e)
	}
	cfg.MultiStatements = true
	db, e := sql.Open("mysql", cfg.FormatDSN())
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	ddl, e := os.ReadFile("../../../../migrations/001_initial.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.ExecContext(ctx, string(ddl)); e != nil {
		t.Fatal(e)
	}
	client := rdb.NewClient(&rdb.Options{Addr: addr})
	defer client.Close()
	projection := &cache.Cache{Client: client, LoadBounds: func(context.Context, string) (cache.Bounds, error) { return cache.Bounds{Min: 100, End: 1000}, nil }}
	repo := &Repository{DB: db, Redis: client, Cache: projection}
	edition := "123"
	user := "9007199254740993"
	ev := func(kind string, f domain.Fact) platform.Event {
		return platform.Event{ID: fmt.Sprint(platform.ID()), Type: kind, Payload: platform.JSON(f)}
	}
	initial := ev("ParticipationInitialized.v1", domain.Fact{Edition: edition, User: user, Created: "100"})
	if e = repo.Consume(ctx, initial); e != nil {
		t.Fatal(e)
	}
	second := ev("FormalDrawCommitted.v1", domain.Fact{Edition: edition, User: user, Draw: "2", Seq: "2", Consumed: "100", Occurred: "202", Total: "105"})
	first := ev("FormalDrawCommitted.v1", domain.Fact{Edition: edition, User: user, Draw: "1", Seq: "1", Consumed: "5", Occurred: "201", Total: "5"})
	if e = repo.CacheUpdate(ctx, second); e != nil {
		t.Fatal(e)
	}
	for _, event := range []platform.Event{second, first, second, first} {
		if e = repo.Consume(ctx, event); e != nil {
			t.Fatal(e)
		}
	}
	var score, seq int64
	if e = db.QueryRowContext(ctx, `SELECT score,last_contiguous_seq FROM ranking_entry WHERE edition_id=? AND user_id=?`, edition, user).Scan(&score, &seq); e != nil {
		t.Fatal(e)
	}
	if score != 105 || seq != 2 {
		t.Fatalf("SQL = %d/%d", score, seq)
	}
	board, e := projection.Board(ctx, edition, user)
	if e != nil || board.Me == nil || board.Me.Score != 105 || board.Me.Reached != 202 {
		t.Fatalf("Redis mismatch: %+v / %v", board.Me, e)
	}
	// Cache failure after SQL commit: replay must retry the delta, not SQL addition.
	third := ev("FormalDrawCommitted.v1", domain.Fact{Edition: edition, User: user, Draw: "3", Seq: "3", Consumed: "7", Occurred: "203", Total: "112"})
	bad := rdb.NewClient(&rdb.Options{Addr: "127.0.0.1:1", MaxRetries: -1})
	projection.Client = bad
	if e = repo.Consume(ctx, third); e == nil {
		t.Fatal("expected cache failure")
	}
	bad.Close()
	projection.Client = client
	if e = repo.Consume(ctx, third); e != nil {
		t.Fatal(e)
	}
	board, e = projection.Board(ctx, edition, user)
	if e != nil || board.Me.Score != 112 {
		t.Fatalf("retry mismatch: %+v %v", board.Me, e)
	}
	// Existing positive SQL history is never replaced with a new partial cache.
	_, e = db.ExecContext(ctx, `INSERT INTO ranking_entry(edition_id,user_id,score,score_reached_at_sec,last_contiguous_seq) VALUES(456,7,500,200,1)`)
	if e != nil {
		t.Fatal(e)
	}
	if e = repo.CacheUpdate(ctx, ev("FormalDrawCommitted.v1", domain.Fact{Edition: "456", User: "7", Draw: "10", Seq: "2", Consumed: "5", Occurred: "201"})); e != nil {
		t.Fatal(e)
	}
	if _, e = projection.Board(ctx, "456", "7"); e == nil {
		t.Fatal("partial old history served")
	}
}
