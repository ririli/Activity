package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/domain"
	cache "activity/services/ranking-service/internal/ranking/infrastructure/redis"
	driver "github.com/go-sql-driver/mysql"
	rdb "github.com/redis/go-redis/v9"
)

func TestSQLAndRedisDelta(t *testing.T) {
	dsn := os.Getenv("RANKING_TEST_MYSQL_DSN")
	addr := os.Getenv("RANKING_TEST_REDIS_ADDR")
	prepared := os.Getenv("RANKING_TEST_PREPARED_SCHEMA") == "1"
	if dsn == "" || addr == "" || (!prepared && os.Getenv("RANKING_TEST_ALLOW_SCHEMA") != "1") {
		t.Skip("requires disposable MySQL and Redis")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg, err := driver.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MultiStatements = true
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if !prepared {
		for _, name := range []string{"001_initial.sql", "002_settlement_resume.sql"} {
			ddl, err := os.ReadFile("../../../../migrations/" + name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = db.ExecContext(ctx, string(ddl)); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Setenv("APP_ENV", fmt.Sprintf("ranking-test-%d", platform.ID()))
	client := rdb.NewClient(&rdb.Options{Addr: addr})
	defer client.Close()
	projection := &cache.Cache{Client: client, LoadBounds: func(context.Context, string) (cache.Bounds, error) {
		return cache.Bounds{Min: 100, End: 1000}, nil
	}}
	repo := &Repository{DB: db, Cache: projection}
	edition := fmt.Sprint(platform.ID())
	defer projection.Retire(context.Background(), edition)
	const user = "9007199254740993"
	init := domain.Fact{Edition: edition, User: user, Created: "100"}
	draw := func(seq, cost, at int) domain.Fact {
		return domain.Fact{Edition: edition, User: user, Draw: fmt.Sprint(platform.ID()), Seq: fmt.Sprint(seq), Consumed: fmt.Sprint(cost), Occurred: fmt.Sprint(at)}
	}
	event := func(f domain.Fact) platform.Event {
		kind := "ParticipationInitialized.v1"
		if f.Draw != "" {
			kind = "FormalDrawCommitted.v1"
		}
		return platform.Event{ID: fmt.Sprint(platform.ID()), Type: kind, Payload: platform.JSON(f)}
	}
	consume := func(f domain.Fact) {
		t.Helper()
		if err := repo.Consume(ctx, event(f)); err != nil {
			t.Fatal(err)
		}
	}
	rebuild := func() {
		t.Helper()
		if err := repo.rebuild(ctx, edition, repo.edition(edition)); err != nil {
			t.Fatal(err)
		}
	}
	expectBoard := func(source string, score, reached int64) {
		t.Helper()
		b, err := repo.Board(ctx, edition, user)
		if err != nil || b.Source != source || b.Me == nil || b.Me.Score != score || b.Me.Reached != reached || b.Me.Rank != 1 {
			t.Fatalf("expected %s score=%d reached=%d, got %+v / %v", source, score, reached, b.Me, err)
		}
	}

	consume(init)
	rebuild()
	first, second := draw(1, 5, 201), draw(2, 100, 202)
	consume(second)
	expectBoard("redis", 100, 202)
	var seq int
	if err := db.QueryRowContext(ctx, `SELECT last_contiguous_seq FROM ranking_entry WHERE edition_id=? AND user_id=?`, edition, user).Scan(&seq); err != nil || seq != 0 {
		t.Fatalf("MySQL advanced past a gap: %d / %v", seq, err)
	}

	// New service instance rebuilds from SQL, including the fact beyond the gap.
	repo = &Repository{DB: db, Cache: projection}
	workerCtx, stopWorker := context.WithCancel(ctx)
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		repo.MaintainCache(workerCtx)
	}()
	// Observe the startup rebuild without registering the edition via a query.
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if value, ok := repo.editions.Load(edition); ok && value.(*liveEdition).generation.Load() != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	stopWorker()
	<-workerDone
	expectBoard("redis", 100, 202)
	consume(second)
	expectBoard("redis", 100, 202)
	consume(first)
	consume(first) // A different delivery ID still deduplicates by draw ID.
	consume(init)
	expectBoard("redis", 105, 202)

	// A live generation must be served even if the SQL connection is closed.
	closedDB, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	closedDB.Close()
	repo.DB = closedDB
	expectBoard("redis", 105, 202)
	third := draw(3, 7, 203)
	if err := repo.Consume(ctx, event(third)); err == nil {
		t.Fatal("event ACKed without SQL commit")
	}
	expectBoard("redis", 112, 203)
	if err := repo.Consume(ctx, event(third)); err == nil {
		t.Fatal("event ACKed on retry without SQL commit")
	}
	expectBoard("redis", 112, 203)
	// A process exit discards the old generation. The unacknowledged draw
	// is applied when the broker redelivers it to the replacement instance.
	repo = &Repository{DB: db, Cache: projection}
	rebuild()
	expectBoard("redis", 105, 202)
	consume(third)
	expectBoard("redis", 112, 203)

	conflict := third
	conflict.Draw = fmt.Sprint(platform.ID())
	if err := repo.Consume(ctx, event(conflict)); err == nil {
		t.Fatal("same sequence with a different draw ID was accepted")
	}
	expectBoard("mysql", 112, 203)
	rebuild()
	expectBoard("redis", 112, 203)

	bad := rdb.NewClient(&rdb.Options{Addr: "127.0.0.1:1", MaxRetries: -1})
	defer bad.Close()
	projection.Client = bad
	fourth := draw(4, 8, 204)
	consume(fourth) // Redis failed, but the transaction must still commit.
	expectBoard("mysql", 120, 204)
	projection.Client = client
	rebuild()
	consume(fourth)
	expectBoard("redis", 120, 204)
	if err := projection.Retire(ctx, edition); err != nil {
		t.Fatal(err)
	}
	expectBoard("mysql", 120, 204)
	rebuild()
	expectBoard("redis", 120, 204)

	// Redis can remain offline throughout cutoff reconciliation and award generation.
	projection.Client = bad
	fifth := draw(5, 9, 205)
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var in map[string]string
		if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var out any
		switch req.URL.Path {
		case "/internal/activity/v1/close":
			out = struct{}{}
		case "/internal/activity/v1/manifest":
			out = []manifestUser{}
			if in["after"] == "" {
				out = []manifestUser{{User: user, Seq: "5", Total: "129", Reached: "205", Created: 100}}
			}
		case "/internal/activity/v1/facts":
			out = []domain.Fact{first, second, third, fourth, fifth}
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer core.Close()
	t.Setenv("ACTIVITY_URL", core.URL)
	for _, phase := range []string{domain.SettlementFrozen, domain.SettlementAwarding, domain.SettlementAwarding} {
		got, err := repo.Advance(ctx, edition)
		if err != nil || got.Status != phase {
			t.Fatalf("expected %s, got %+v / %v", phase, got, err)
		}
	}
	var awards int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ranking_award WHERE edition_id=?`, edition).Scan(&awards); err != nil || awards != 2 {
		t.Fatalf("expected two unique awards, got %d / %v", awards, err)
	}
	rows, err := db.QueryContext(ctx, `SELECT CAST(award_id AS CHAR) FROM ranking_award WHERE edition_id=?`, edition)
	if err != nil {
		t.Fatal(err)
	}
	var awardIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		awardIDs = append(awardIDs, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range awardIDs {
		receipt := platform.Event{Type: "RankingAwardGranted.v1", Payload: platform.JSON(map[string]string{
			"award_id": id, "edition_id": edition, "user_id": user,
		})}
		for i := 0; i < 2; i++ {
			if err := repo.Consume(ctx, receipt); err != nil {
				t.Fatal(err)
			}
		}
	}
	if got, err := repo.Advance(ctx, edition); err != nil || got.Status != domain.SettlementCompleted {
		t.Fatalf("expected completed after receipts, got %+v / %v", got, err)
	}
	// Make the live projection distinct to prove the finalized read uses the snapshot.
	if _, err := db.ExecContext(ctx, `UPDATE ranking_entry SET score=1 WHERE edition_id=?`, edition); err != nil {
		t.Fatal(err)
	}
	expectBoard("mysql", 129, 205)
	projection.Client = client
	rebuild() // finalized editions only retire the cache
	expectBoard("mysql", 129, 205)
}
