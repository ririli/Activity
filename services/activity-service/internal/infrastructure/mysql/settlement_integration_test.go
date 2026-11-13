package mysql

import (
	"activity/pkg/platform"
	app "activity/services/activity-service/internal/application"
	"activity/services/activity-service/internal/fault"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestSettlementClaimsHaveUniqueOwners(t *testing.T) {
	t.Setenv("NODE_ID", "same-process")
	a, b := taskOwner(), taskOwner()
	if a == b || len(a) > 64 || len(b) > 64 {
		t.Fatalf("owners %q %q", a, b)
	}
}

func TestRankingAwardReplayGrantsOnce(t *testing.T) {
	dsn := os.Getenv("ACTIVITY_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("requires disposable initialized activity schema")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	user, award := platform.ID(), platform.ID()
	if _, err := db.ExecContext(ctx, `INSERT INTO wallet(user_id) VALUES(?)`, user); err != nil {
		t.Fatal(err)
	}
	s := app.Service{UoW: UnitOfWork{DB: db}, Runtime: app.Runtime{NewID: platform.ID}}
	cmd := app.RankingAwardCommand{User: fmt.Sprint(user), Edition: "1", Award: fmt.Sprint(award), Item: "1323", Name: "title", Quantity: 1}
	for i := 0; i < 2; i++ {
		if err := s.GrantRankingAward(ctx, cmd); err != nil {
			t.Fatal(err)
		}
	}
	var grants int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reward_grant
		WHERE user_id=? AND source_type='RANKING' AND source_id=?`, user, award).Scan(&grants); err != nil || grants != 1 {
		t.Fatalf("expected one grant, got %d / %v", grants, err)
	}
	var receipts, grantIDs int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*),COUNT(DISTINCT JSON_UNQUOTE(JSON_EXTRACT(payload,'$.grant_id')))
		FROM outbox_event WHERE event_type='RankingAwardGranted.v1' AND aggregate_id=?`, award).Scan(&receipts, &grantIDs); err != nil || receipts != 2 || grantIDs != 1 {
		t.Fatalf("expected two receipts with one grant ID, got %d/%d / %v", receipts, grantIDs, err)
	}
}

func TestStaleSettlementCannotFailEdition(t *testing.T) {
	dsn := os.Getenv("ACTIVITY_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("requires disposable initialized activity schema")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	edition := platform.ID()
	_, err = db.ExecContext(ctx, `INSERT INTO activity_edition(edition_id,activity_code,name,starts_at,ends_at,status,active_rule_version,settlement_status) VALUES(?,'review','review',UTC_TIMESTAMP()-INTERVAL 2 HOUR,UTC_TIMESTAMP()-INTERVAL 1 HOUR,'ENDED',1,'RUNNING')`, edition)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO settlement_task(edition_id,step,partition_no,status,lease_owner,lease_until) VALUES(?,'01_CLOSE_EDITION',-1,'RUNNING','new-owner',UTC_TIMESTAMP()+INTERVAL 60 SECOND)`, edition)
	if err != nil {
		t.Fatal(err)
	}
	s := TaskStore{DB: db}
	task := app.SettlementTask{Edition: fmt.Sprint(edition), Step: "01_CLOSE_EDITION", Partition: -1, Owner: "old-owner"}
	if err = s.Fail(ctx, task, "OLD_FAILURE"); err == nil {
		t.Fatal("stale failure accepted")
	}
	if err = s.Release(ctx, task, []byte(`{}`), time.Second); err == nil {
		t.Fatal("stale release accepted")
	}
	var status string
	if err = db.QueryRowContext(ctx, `SELECT settlement_status FROM activity_edition WHERE edition_id=?`, edition).Scan(&status); err != nil || status != "RUNNING" {
		t.Fatalf("edition corrupted: %s %v", status, err)
	}
	task.Owner = "new-owner"
	if err = s.Fail(ctx, task, "REAL_FAILURE"); err == nil || err.Error() != "REAL_FAILURE" {
		t.Fatalf("owned failure: %v", err)
	}
	if err = db.QueryRowContext(ctx, `SELECT settlement_status FROM activity_edition WHERE edition_id=?`, edition).Scan(&status); err != nil || status != "FAILED" {
		t.Fatalf("failure not committed: %s %v", status, err)
	}
}

func TestMutationSeesCommitAfterEarlierRead(t *testing.T) {
	dsn := os.Getenv("ACTIVITY_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("requires disposable initialized activity schema")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	user, order := platform.ID(), platform.ID()
	if _, err = db.ExecContext(ctx, `INSERT INTO wallet(user_id) VALUES(?)`, user); err != nil {
		t.Fatal(err)
	}
	err = (UnitOfWork{DB: db}).Within(ctx, app.Scope{User: user, Edition: 1}, func(s app.Session) error {
		if _, _, _, _, _, err := s.Assets.FindPurchase("snapshot-test"); !errors.Is(err, fault.ErrNotFound) {
			return fmt.Errorf("first read: %v", err)
		}
		// A different transaction commits between an earlier read and the wallet
		// lock, just as an independently running command/consumer can do.
		if _, err := db.ExecContext(ctx, `INSERT INTO purchase_order(order_id,user_id,edition_id,request_id,purchase_type,product_code,gold_cost,petals_granted,rule_version) VALUES(?,?,1,'snapshot-test','SHOP','5',300,5,1)`, order, user); err != nil {
			return err
		}
		if err := s.Assets.LockWallet(); err != nil {
			return err
		}
		id, _, _, _, _, err := s.Assets.FindPurchase("snapshot-test")
		if err != nil {
			return fmt.Errorf("read after lock missed committed order: %w", err)
		}
		if id != order {
			return fmt.Errorf("wrong order %d", id)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
