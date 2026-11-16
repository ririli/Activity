package platform

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"
)

const (
	snowflakeEpoch = 1767225600000
	maxIDMillis    = 1<<41 - 1
	reservationMS  = 250
	clockWait      = 10 * time.Millisecond
	allocationWait = 250 * time.Millisecond
)

var ErrIDUnavailable = errors.New("snowflake ID unavailable")

// IDGenerator issues IDs only from timestamp ranges committed by its service's
// database. A crashed process loses the unused part of its range.
type IDGenerator struct {
	node      int64
	wait      time.Duration
	gate      chan struct{}
	now       func() time.Time
	reserve   func(context.Context, int64) (start, end int64, err error)
	sequence  idSequence
	until     int64
	nextRange int64
}

// OpenIDs uses a separate connection pool so renewing a range can still make
// progress when all business connections are occupied by transactions.
func OpenIDs() (*IDGenerator, func(), error) {
	node, err := strconv.ParseInt(Env("NODE_ID", ""), 10, 10)
	if err != nil || node < 0 || node > 1023 {
		return nil, nil, errors.New("NODE_ID must be configured between 0 and 1023")
	}
	wait, err := time.ParseDuration(Env("SNOWFLAKE_CLOCK_WAIT", clockWait.String()))
	if err != nil || wait <= 0 {
		return nil, nil, errors.New("SNOWFLAKE_CLOCK_WAIT must be a positive duration")
	}
	db := openMySQL()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	g := newIDGenerator(node, time.Now, func(ctx context.Context, ms int64) (int64, int64, error) {
		return reserveIDRange(ctx, db, node, ms)
	})
	g.wait = wait
	return g, func() { _ = db.Close() }, nil
}

func newIDGenerator(node int64, now func() time.Time, reserve func(context.Context, int64) (int64, int64, error)) *IDGenerator {
	return &IDGenerator{node: node, wait: clockWait, now: now, reserve: reserve, gate: make(chan struct{}, 1),
		sequence: idSequence{lastMS: -1}, until: -1}
}

func (g *IDGenerator) Next(ctx context.Context) (uint64, error) { return g.next(ctx, false) }

// NextUser preserves the rotation across the four user_id % 4 shards.
func (g *IDGenerator) NextUser(ctx context.Context) (uint64, error) { return g.next(ctx, true) }

func (g *IDGenerator) next(ctx context.Context, user bool) (uint64, error) {
	waiting := false
	for {
		select {
		case g.gate <- struct{}{}:
		case <-ctx.Done():
			return 0, fmt.Errorf("%w: %v", ErrIDUnavailable, ctx.Err())
		}
		id, err := g.tryNext(ctx, user)
		<-g.gate
		if id != 0 || err != nil {
			return id, err
		}
		if !waiting {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, g.wait)
			defer cancel()
			waiting = true
		}
		timer := time.NewTimer(time.Millisecond)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return 0, fmt.Errorf("%w: %v", ErrIDUnavailable, ctx.Err())
		}
	}
}

// A zero ID with no error means the process must wait for the wall clock.
func (g *IDGenerator) tryNext(ctx context.Context, user bool) (uint64, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrIDUnavailable, err)
	}
	ms := g.now().UnixMilli() - snowflakeEpoch
	if ms <= 0 || ms > maxIDMillis {
		return 0, fmt.Errorf("%w: timestamp outside the supported epoch", ErrIDUnavailable)
	}
	if ms < g.sequence.lastMS {
		g.nextRange = max(g.nextRange, g.sequence.lastMS+1)
	}
	if ms < g.nextRange {
		return 0, nil
	}
	if ms > g.until {
		reserveCtx, cancel := context.WithTimeout(ctx, allocationWait)
		start, end, err := g.reserve(reserveCtx, ms)
		cancel()
		if err != nil {
			return 0, fmt.Errorf("%w: reserve timestamp range: %v", ErrIDUnavailable, err)
		}
		g.nextRange = start
		if start > ms {
			return 0, nil
		}
		g.until = end
		// The database round trip may have crossed a millisecond or a rollback.
		ms = g.now().UnixMilli() - snowflakeEpoch
		if ms < start || ms > end || ctx.Err() != nil {
			return 0, nil
		}
	}
	return g.sequence.allocate(ms, g.node, user), nil
}

type idSequence struct{ lastMS, next, userSlot int64 }

func (s *idSequence) allocate(ms, node int64, user bool) uint64 {
	if ms != s.lastMS {
		s.lastMS, s.next = ms, 0
	}
	seq := s.next
	if user {
		seq += (s.userSlot - seq) & 3
	}
	if seq >= 4096 {
		return 0
	}
	s.next = seq + 1
	if user {
		s.userSlot = (s.userSlot + 1) % 4
	}
	return uint64(ms<<22 | node<<12 | seq)
}

func reserveIDRange(ctx context.Context, db *sql.DB, node, ms int64) (int64, int64, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO snowflake_reservation(node_id,reserved_until_ms) VALUES(?,0)
		ON DUPLICATE KEY UPDATE node_id=node_id`, node); err != nil {
		return 0, 0, err
	}
	var previous int64
	if err := tx.QueryRowContext(ctx, `SELECT reserved_until_ms FROM snowflake_reservation WHERE node_id=? FOR UPDATE`, node).Scan(&previous); err != nil {
		return 0, 0, err
	}
	if previous >= ms {
		return previous + 1, 0, nil
	}
	end := min(ms+reservationMS-1, maxIDMillis)
	if _, err := tx.ExecContext(ctx, `UPDATE snowflake_reservation SET reserved_until_ms=? WHERE node_id=?`, end, node); err != nil {
		return 0, 0, err
	}
	return ms, end, tx.Commit()
}
