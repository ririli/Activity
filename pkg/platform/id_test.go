package platform

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestUserIDsRotateAcrossMilliseconds(t *testing.T) {
	var sequence idSequence
	seen := make(map[uint64]bool)
	var allocated uint64
	for ms := int64(100); ms < 124; ms++ {
		// Alternate a quiet millisecond and bursts with ordinary IDs interleaved.
		for range 1 + ms%2*2 {
			ordinary := sequence.allocate(ms, 9, false)
			user := sequence.allocate(ms, 9, true)
			if user%4 != allocated%4 || user>>22 != uint64(ms) || user>>12&1023 != 9 {
				t.Fatalf("unexpected user ID %d at allocation %d", user, allocated)
			}
			if user <= ordinary || seen[user] || seen[ordinary] {
				t.Fatalf("IDs must be unique and increasing: %d, %d", ordinary, user)
			}
			seen[ordinary], seen[user] = true, true
			allocated++
		}
	}
}

func TestUserIDSequenceDoesNotOverflowNodeBits(t *testing.T) {
	sequence := idSequence{lastMS: 100, next: 4095, userSlot: 0}
	if id := sequence.allocate(100, 9, true); id != 0 {
		t.Fatalf("expected sequence exhaustion, got %d", id)
	}
	if user := sequence.allocate(101, 9, true); user%4 != 0 || user>>12&1023 != 9 {
		t.Fatalf("failed allocation must preserve the next user slot: %d", user)
	}
}

func TestIDGeneratorClockRollbackAndSequenceExhaustion(t *testing.T) {
	var ms atomic.Int64
	ms.Store(100)
	reserve := func(_ context.Context, at int64) (int64, int64, error) { return at, at + 249, nil }
	g := newIDGenerator(9, func() time.Time { return time.UnixMilli(snowflakeEpoch + ms.Load()) }, reserve)
	ctx := context.Background()
	first, err := g.Next(ctx)
	if err != nil || first>>22 != 100 {
		t.Fatalf("first ID = %d, %v", first, err)
	}
	ms.Store(99)
	if id, err := g.Next(ctx); id != 0 || !errors.Is(err, ErrIDUnavailable) {
		t.Fatalf("rollback issued %d, %v", id, err)
	}
	ms.Store(100)
	if id, err := g.Next(ctx); id != 0 || !errors.Is(err, ErrIDUnavailable) {
		t.Fatalf("reused rolled-back millisecond: %d, %v", id, err)
	}
	ms.Store(101)
	second, err := g.Next(ctx)
	if err != nil || second <= first || second>>22 != 101 {
		t.Fatalf("resumed ID = %d, %v", second, err)
	}
	g.sequence.next = 4096
	if id, err := g.Next(ctx); id != 0 || !errors.Is(err, ErrIDUnavailable) {
		t.Fatalf("sequence exhaustion issued %d, %v", id, err)
	}
	ms.Store(102)
	if id, err := g.Next(ctx); err != nil || id>>22 != 102 {
		t.Fatalf("next millisecond: %d, %v", id, err)
	}
}

func TestIDGeneratorWaitsForClockRecovery(t *testing.T) {
	var reads int
	g := newIDGenerator(9, func() time.Time {
		reads++
		ms := int64(101)
		if reads <= 2 {
			ms = 99
		}
		return time.UnixMilli(snowflakeEpoch + ms)
	}, func(_ context.Context, at int64) (int64, int64, error) { return at, at + 249, nil })
	g.sequence.lastMS, g.until = 100, 200
	g.wait = 100 * time.Millisecond
	if id, err := g.Next(context.Background()); err != nil || id>>22 != 101 {
		t.Fatalf("short rollback did not recover: %d, %v", id, err)
	}
}

func TestMySQLSnowflakeReservationSurvivesRestart(t *testing.T) {
	dsn := os.Getenv("PLATFORM_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("requires a disposable MySQL schema with the Snowflake migration")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const node = 1023
	var exists bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM snowflake_reservation WHERE node_id=?)`, node).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("test node is already in use")
	}
	t.Cleanup(func() {
		cleanup, err := sql.Open("mysql", dsn)
		if err != nil {
			t.Error(err)
			return
		}
		defer cleanup.Close()
		if _, err := cleanup.Exec(`DELETE FROM snowflake_reservation WHERE node_id=?`, node); err != nil {
			t.Error(err)
		}
	})
	var clock atomic.Int64
	clock.Store(100)
	now := func() time.Time { return time.UnixMilli(snowflakeEpoch + clock.Load()) }
	reserve := func(ctx context.Context, at int64) (int64, int64, error) { return reserveIDRange(ctx, db, node, at) }
	a, b := newIDGenerator(node, now, reserve), newIDGenerator(node, now, reserve)
	var wg sync.WaitGroup
	type result struct {
		id  uint64
		err error
	}
	issued := make(chan result, 2)
	for _, g := range []*IDGenerator{a, b} {
		wg.Go(func() { id, err := g.Next(ctx); issued <- result{id, err} })
	}
	wg.Wait()
	close(issued)
	successes := 0
	for outcome := range issued {
		if outcome.err == nil && outcome.id>>22 == 100 {
			successes++
			continue
		}
		if outcome.id != 0 || !errors.Is(outcome.err, ErrIDUnavailable) {
			t.Fatalf("concurrent reservation: %+v", outcome)
		}
	}
	if successes != 1 {
		t.Fatalf("issued %d IDs from competing allocators", successes)
	}
	clock.Store(350)
	restarted := newIDGenerator(node, now, reserve)
	if id, err := restarted.Next(ctx); err != nil || id>>22 != 350 {
		t.Fatalf("restarted allocation = %d, %v", id, err)
	}
}
