package platform

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
)

// Requires a disposable initialized schema, with service dispatchers stopped.
func TestOutboxFailureDoesNotStarveLaterEvents(t *testing.T) {
	dsn := os.Getenv("PLATFORM_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("requires disposable MySQL with outbox_event")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	first, second := fmt.Sprint(ID()), fmt.Sprint(ID())
	for _, id := range []string{first, second} {
		_, err = db.ExecContext(ctx, `INSERT INTO outbox_event(event_id,event_type,aggregate_id,payload) VALUES(?,'ReviewTest.v1',1,JSON_OBJECT())`, id)
		if err != nil {
			t.Fatal(err)
		}
	}
	failed := errors.New("broker unavailable")
	found, err := dispatchNext(ctx, db, func(context.Context, Event) error { return failed })
	if !found || !errors.Is(err, failed) {
		t.Fatalf("%v %v", found, err)
	}
	var retries, deferred int
	if err = db.QueryRowContext(ctx, `SELECT retry_count,next_retry_at>UTC_TIMESTAMP(6) FROM outbox_event WHERE event_id=? AND status='PENDING'`, first).Scan(&retries, &deferred); err != nil || retries != 1 || deferred != 1 {
		t.Fatalf("retry not persisted: %d %d %v", retries, deferred, err)
	}
	published := []string{}
	publish := func(_ context.Context, ev Event) error { published = append(published, ev.ID); return nil }
	found, err = dispatchNext(ctx, db, publish)
	if err != nil || !found || len(published) != 1 || published[0] != second {
		t.Fatalf("later event blocked: %v %v %v", found, err, published)
	}
	if _, err = db.ExecContext(ctx, `UPDATE outbox_event SET next_retry_at=UTC_TIMESTAMP(6) WHERE event_id=?`, first); err != nil {
		t.Fatal(err)
	}
	found, err = dispatchNext(ctx, db, publish)
	if err != nil || !found || len(published) != 2 || published[1] != first {
		t.Fatalf("retry failed: %v %v %v", found, err, published)
	}
	found, err = dispatchNext(ctx, db, publish)
	if err != nil || found || len(published) != 2 {
		t.Fatalf("published event repeated: %v %v", found, err)
	}
}
