package mysql

import (
	"activity/pkg/hourly"
	"activity/pkg/platform"
	"activity/pkg/platform/testid"
	app "activity/services/activity-service/internal/application"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

func TestHourlySealingAndGrantReplay(t *testing.T) {
	dsn := os.Getenv("ACTIVITY_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("requires disposable initialized activity schema")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	edition, user := testid.Next(), testid.Next()
	cutoff := time.Now().UTC().Truncate(time.Hour)
	window := hourly.Window{Edition: fmt.Sprint(edition), Cutoff: cutoff.Unix()}
	shard, lane := hourly.Shard(user), hourly.Lane(user)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO wallet(user_id,petal_balance) VALUES(?,50)`, user)
	exec(`INSERT INTO activity_edition(edition_id,activity_code,name,starts_at,ends_at,status,active_rule_version,hourly_ranking_enabled)
		VALUES(?,'hourly-test','hourly-test',?,?,'ENDED',1,TRUE)`, edition, cutoff.Add(-time.Minute), cutoff.Add(time.Hour))
	for part := range hourly.Shards {
		exec(`INSERT INTO hourly_ranking_window(edition_id,cutoff_sec,shard_no) VALUES(?,?,?)`, edition, window.Cutoff, part)
	}
	queries := &Queries{DB: db}
	scope := app.Scope{User: user, Edition: edition}
	rolledBack := errors.New("rollback test")
	appendFact := func(tx *sql.Tx, code string) error {
		var sealed sql.NullTime
		if err := tx.QueryRowContext(ctx, `SELECT sealed_at FROM hourly_ranking_window WHERE edition_id=? AND cutoff_sec=? AND shard_no=? FOR SHARE`, edition, window.Cutoff, shard).Scan(&sealed); err != nil {
			return err
		}
		o := &outbox{ctx: ctx, tx: tx, position: hourly.Position{Cutoff: window.Cutoff, Shard: shard, Lane: lane}}
		err := o.Append(app.Event{ID: fmt.Sprint(testid.Next()), Type: code, Payload: platform.JSON(map[string]string{"user_id": fmt.Sprint(user)})}, user, edition)
		if err != nil {
			return err
		}
		return o.flush(scope)
	}
	commit := func(commit bool, kind string) (*sql.Tx, error) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		if err := appendFact(tx, kind); err != nil {
			tx.Rollback()
			return nil, err
		}
		if !commit {
			tx.Rollback()
			return nil, rolledBack
		}
		return tx, nil
	}
	if _, err := commit(false, "FormalDrawCommitted.v1"); !errors.Is(err, rolledBack) {
		t.Fatal(err)
	}
	late, err := commit(true, "ParticipationInitialized.v1")
	if err != nil {
		t.Fatal(err)
	}
	defer late.Rollback()
	type result struct {
		mark hourly.Watermark
		err  error
	}
	sealed := make(chan result, 1)
	go func() { mark, err := queries.SealHourly(ctx, window); sealed <- result{mark, err} }()
	select {
	case got := <-sealed:
		t.Fatalf("sealed an in-flight transaction: %+v", got)
	case <-time.After(100 * time.Millisecond):
	}
	if err := late.Commit(); err != nil {
		t.Fatal(err)
	}
	got := <-sealed
	if got.err != nil || got.mark[shard][lane] != 1 {
		t.Fatalf("bad watermark: %+v", got)
	}
	mark, err := queries.SealHourly(ctx, window)
	if err != nil || mark != got.mark {
		t.Fatalf("seal replay: %v %v", mark, err)
	}
	page, err := queries.ReplayHourly(ctx, hourly.Replay{Window: window, Shard: shard, Lane: lane, Through: 1})
	if err != nil || len(page) != 1 || page[0].Type != "ParticipationInitialized.v1" {
		t.Fatalf("replay: %+v %v", page, err)
	}
	var payload struct{ Hourly hourly.Position }
	if err := json.Unmarshal(page[0].Payload, &payload); err != nil || payload.Hourly.Seq != 1 {
		t.Fatalf("replay position: %+v %v", payload, err)
	}
	service := app.HourlyService{Queries: queries, UoW: UnitOfWork{DB: db, NewID: testid.Generate}, NewID: testid.Generate}
	grant := hourly.Grant{Barrier: hourly.Barrier{Window: window, Settlement: fmt.Sprint(testid.Next())}, User: fmt.Sprint(user), Rank: 1}
	var calls sync.WaitGroup
	receipts := make([]hourly.Receipt, 8)
	for i := range receipts {
		calls.Go(func() {
			var err error
			receipts[i], err = service.Grant(ctx, grant)
			if err != nil {
				t.Error(err)
			}
		})
	}
	calls.Wait()
	for _, receipt := range receipts {
		if receipt.GrantID == "" || receipt != receipts[0] {
			t.Fatalf("different receipt: %+v", receipts)
		}
	}
	var balance, grants, ledgers int
	query := fmt.Sprintf(`SELECT petal_balance,
		(SELECT COUNT(*) FROM reward_grant WHERE user_id=w.user_id AND source_type='HOURLY_RANKING' AND quantity=1000),
		(SELECT COUNT(*) FROM %s WHERE user_id=w.user_id AND reason='HOURLY_RANKING_AWARD' AND delta=1000 AND balance_after=1050)
		FROM wallet w WHERE user_id=?`, platform.Shard("wallet_ledger", user))
	if err := db.QueryRowContext(ctx, query, user).Scan(&balance, &grants, &ledgers); err != nil || balance != 1050 || grants != 1 || ledgers != 1 {
		t.Fatalf("balance=%d grants=%d ledgers=%d error=%v", balance, grants, ledgers, err)
	}
}
