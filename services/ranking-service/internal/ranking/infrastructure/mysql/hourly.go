package mysql

import (
	"activity/pkg/hourly"
	"activity/pkg/platform"
	app "activity/services/ranking-service/internal/ranking/application"
	"activity/services/ranking-service/internal/ranking/domain"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type HourlyStore struct{ DB *sql.DB }

func (s HourlyStore) ScheduleHourly(ctx context.Context, editions []hourly.Edition) (time.Time, error) {
	var now, next time.Time
	if err := s.DB.QueryRowContext(ctx, `SELECT UTC_TIMESTAMP(6)`).Scan(&now); err != nil {
		return next, err
	}
	for _, edition := range editions {
		if !canonicalID(edition.ID) || !edition.Start.Before(edition.End) {
			return next, fmt.Errorf("invalid hourly edition")
		}
		var last sql.NullInt64
		if err := s.DB.QueryRowContext(ctx, `SELECT MAX(cutoff_sec) FROM hourly_settlement WHERE edition_id=?`, edition.ID).Scan(&last); err != nil {
			return next, err
		}
		at := hourly.NextCutoff(edition.Start)
		if last.Valid {
			at = time.Unix(last.Int64, 0).UTC().Add(time.Hour)
		}
		// Limit each discovery pass so a long outage cannot starve other editions.
		for range 48 {
			if !at.Before(edition.End) || at.After(now) {
				break
			}
			if err := s.enqueueBarrier(ctx, edition.ID, at.Unix(), last); err != nil {
				return next, err
			}
			last = sql.NullInt64{Int64: at.Unix(), Valid: true}
			at = at.Add(time.Hour)
		}
		if at.Before(edition.End) && (next.IsZero() || at.Before(next)) {
			next = at
		}
	}
	// A broker confirmation proves publication, while the task tracks receipt.
	_, err := s.DB.ExecContext(ctx, `UPDATE outbox_event o JOIN hourly_settlement h ON h.settlement_id=o.event_id
		SET o.status='PENDING', o.next_retry_at=UTC_TIMESTAMP(6)
		WHERE h.status='PENDING_BARRIER' AND o.event_type=? AND o.status='PUBLISHED'
		AND o.published_at<=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 30 SECOND)`, hourly.BarrierType)
	return next, err
}

func (s HourlyStore) enqueueBarrier(ctx context.Context, edition string, cutoff int64, previous sql.NullInt64) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	id := platform.ID()
	result, err := tx.ExecContext(ctx, `INSERT INTO hourly_settlement(settlement_id,edition_id,cutoff_sec,previous_cutoff_sec) VALUES(?,?,?,?)
		ON DUPLICATE KEY UPDATE cutoff_sec=hourly_settlement.cutoff_sec`, id, edition, cutoff, previous)
	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if inserted == 1 {
		barrier := hourly.Barrier{Window: hourly.Window{Edition: edition, Cutoff: cutoff}, Settlement: fmt.Sprint(id)}
		if _, err := tx.ExecContext(ctx, `INSERT INTO outbox_event(event_id,event_type,aggregate_id,payload) VALUES(?,?,?,?)`, id, hourly.BarrierType, id, platform.JSON(barrier)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s HourlyStore) ReceiveBarrier(ctx context.Context, ev platform.Event) error {
	var b hourly.Barrier
	if ev.Type != hourly.BarrierType || json.Unmarshal(ev.Payload, &b) != nil || ev.ID != b.Settlement ||
		!canonicalID(b.Settlement) || !canonicalID(b.Edition) || b.Cutoff <= 0 || b.Cutoff%3600 != 0 {
		return platform.Fail(409, "INVALID_HOURLY_BARRIER")
	}
	res, err := s.DB.ExecContext(ctx, `UPDATE hourly_settlement SET status='WAITING_WATERMARK', barrier_received_at=UTC_TIMESTAMP(6), next_retry_at=UTC_TIMESTAMP(6)
		WHERE settlement_id=? AND edition_id=? AND cutoff_sec=? AND status='PENDING_BARRIER'`, b.Settlement, b.Edition, b.Cutoff)
	if err != nil {
		return err
	}
	updated, err := res.RowsAffected()
	if err != nil || updated == 1 {
		return err
	}
	var exists bool
	err = s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM hourly_settlement WHERE settlement_id=? AND edition_id=? AND cutoff_sec=?)`,
		b.Settlement, b.Edition, b.Cutoff).Scan(&exists)
	if err == nil && !exists {
		return platform.Fail(409, "UNKNOWN_HOURLY_BARRIER")
	}
	return err
}

func (s HourlyStore) ClaimHourly(ctx context.Context) (*app.HourlyTask, error) {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var t app.HourlyTask
	var raw []byte
	err = tx.QueryRowContext(ctx, `SELECT CAST(h.settlement_id AS CHAR),CAST(h.edition_id AS CHAR),h.cutoff_sec,h.watermark,h.retry_count
		FROM hourly_settlement h WHERE h.status='WAITING_WATERMARK' AND h.next_retry_at<=UTC_TIMESTAMP(6)
		AND (h.lease_until IS NULL OR h.lease_until<=UTC_TIMESTAMP(6))
		AND (h.previous_cutoff_sec IS NULL OR EXISTS (SELECT 1 FROM hourly_settlement p WHERE p.edition_id=h.edition_id AND p.cutoff_sec=h.previous_cutoff_sec AND p.frozen_at IS NOT NULL))
		ORDER BY h.cutoff_sec,h.edition_id LIMIT 1 FOR UPDATE OF h SKIP LOCKED`).Scan(&t.Settlement, &t.Edition, &t.Cutoff, &raw, &t.Retries)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(raw) != 0 {
		t.Watermark = new(hourly.Watermark)
		if err := json.Unmarshal(raw, t.Watermark); err != nil {
			return nil, err
		}
	}
	t.Owner = leaseOwner()
	if _, err := tx.ExecContext(ctx, `UPDATE hourly_settlement SET lease_owner=?,lease_until=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 60 SECOND) WHERE settlement_id=?`, t.Owner, t.Settlement); err != nil {
		return nil, err
	}
	return &t, tx.Commit()
}

func requireHourlyLease(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err == nil && n != 1 {
		return errors.New("hourly lease lost")
	}
	return err
}

func (s HourlyStore) SaveHourlyWatermark(ctx context.Context, t app.HourlyTask, mark hourly.Watermark) error {
	return requireHourlyLease(s.DB.ExecContext(ctx, `UPDATE hourly_settlement SET watermark=? WHERE settlement_id=? AND lease_owner=? AND lease_until>UTC_TIMESTAMP(6) AND status='WAITING_WATERMARK'`, platform.JSON(mark), t.Settlement, t.Owner))
}

func (s HourlyStore) PreviousHourlyWinners(ctx context.Context, t app.HourlyTask) ([]domain.Entry, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT CAST(a.user_id AS CHAR),a.score,a.reached_at_sec,a.final_rank FROM hourly_settlement h
		JOIN hourly_award a ON a.edition_id=h.edition_id AND a.cutoff_sec=h.previous_cutoff_sec WHERE h.settlement_id=? ORDER BY a.final_rank`, t.Settlement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []domain.Entry
	for rows.Next() {
		var e domain.Entry
		if err := rows.Scan(&e.User, &e.Score, &e.Reached, &e.Rank); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s HourlyStore) FreezeHourly(ctx context.Context, t app.HourlyTask, entries []domain.Entry) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	status := "AWARDING"
	if len(entries) == 0 {
		status = "COMPLETED"
	}
	err = requireHourlyLease(tx.ExecContext(ctx, `UPDATE hourly_settlement SET status=?,frozen_at=UTC_TIMESTAMP(6),
		completed_at=IF(?='COMPLETED',UTC_TIMESTAMP(6),NULL),lease_owner=NULL,lease_until=NULL
		WHERE settlement_id=? AND status='WAITING_WATERMARK' AND lease_owner=? AND lease_until>UTC_TIMESTAMP(6)`, status, status, t.Settlement, t.Owner))
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		values := make([]string, len(entries))
		args := make([]any, 0, len(entries)*8)
		for i, entry := range entries {
			values[i] = "(?,?,?,?,?,?,?,?)"
			args = append(args, t.Edition, t.Cutoff, entry.User, t.Settlement, entry.Rank, entry.Score, entry.Reached, hourly.Petals)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO hourly_award(edition_id,cutoff_sec,user_id,settlement_id,final_rank,score,reached_at_sec,quantity) VALUES `+joinValues(values), args...); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s HourlyStore) RetryHourly(ctx context.Context, t app.HourlyTask, failed bool) error {
	delay, count := 0, 0
	if failed {
		delay, count = 1<<min(t.Retries, 6), 1
	}
	return requireHourlyLease(s.DB.ExecContext(ctx, `UPDATE hourly_settlement SET lease_owner=NULL,lease_until=NULL,
		next_retry_at=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL ? SECOND),retry_count=retry_count+?
		WHERE settlement_id=? AND lease_owner=? AND status='WAITING_WATERMARK'`, delay, count, t.Settlement, t.Owner))
}

func (s HourlyStore) HourlyRetirements(ctx context.Context) ([]hourly.Window, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT CAST(edition_id AS CHAR),cutoff_sec FROM hourly_settlement
		WHERE redis_retired_at IS NULL AND frozen_at IS NOT NULL ORDER BY frozen_at LIMIT 8`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var windows []hourly.Window
	for rows.Next() {
		var w hourly.Window
		if err := rows.Scan(&w.Edition, &w.Cutoff); err != nil {
			return nil, err
		}
		windows = append(windows, w)
	}
	return windows, rows.Err()
}

func (s HourlyStore) FinishHourlyRetirement(ctx context.Context, w hourly.Window) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE hourly_settlement SET redis_retired_at=UTC_TIMESTAMP(6)
		WHERE edition_id=? AND cutoff_sec=? AND redis_retired_at IS NULL AND frozen_at IS NOT NULL`, w.Edition, w.Cutoff)
	return err
}

func (s HourlyStore) ClaimHourlyAward(ctx context.Context) (*app.HourlyAward, error) {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var a app.HourlyAward
	err = tx.QueryRowContext(ctx, `SELECT CAST(settlement_id AS CHAR),CAST(edition_id AS CHAR),cutoff_sec,CAST(user_id AS CHAR),final_rank,retry_count
		FROM hourly_award WHERE delivered_at IS NULL AND next_retry_at<=UTC_TIMESTAMP(6) AND (lease_until IS NULL OR lease_until<=UTC_TIMESTAMP(6))
		ORDER BY next_retry_at,cutoff_sec,final_rank LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&a.Settlement, &a.Edition, &a.Cutoff, &a.User, &a.Rank, &a.Retries)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.Owner = leaseOwner()
	if _, err := tx.ExecContext(ctx, `UPDATE hourly_award SET lease_owner=?,lease_until=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 30 SECOND) WHERE edition_id=? AND cutoff_sec=? AND user_id=?`, a.Owner, a.Edition, a.Cutoff, a.User); err != nil {
		return nil, err
	}
	return &a, tx.Commit()
}

func (s HourlyStore) FinishHourlyAward(ctx context.Context, a app.HourlyAward, receipt string) error {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Lock the parent first so concurrent final confirmations cannot both miss completion.
	var status string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM hourly_settlement WHERE settlement_id=? FOR UPDATE`, a.Settlement).Scan(&status); err != nil {
		return err
	}
	if status != "AWARDING" {
		return errors.New("hourly award settlement is not awarding")
	}
	err = requireHourlyLease(tx.ExecContext(ctx, `UPDATE hourly_award SET grant_id=?,delivered_at=UTC_TIMESTAMP(6),lease_owner=NULL,lease_until=NULL
		WHERE edition_id=? AND cutoff_sec=? AND user_id=? AND lease_owner=? AND lease_until>UTC_TIMESTAMP(6) AND delivered_at IS NULL`, receipt, a.Edition, a.Cutoff, a.User, a.Owner))
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE hourly_settlement SET status='COMPLETED',completed_at=UTC_TIMESTAMP(6)
		WHERE settlement_id=? AND NOT EXISTS (SELECT 1 FROM hourly_award WHERE settlement_id=? AND delivered_at IS NULL)`, a.Settlement, a.Settlement); err != nil {
		return err
	}
	return tx.Commit()
}

func (s HourlyStore) RetryHourlyAward(ctx context.Context, a app.HourlyAward) error {
	return requireHourlyLease(s.DB.ExecContext(ctx, `UPDATE hourly_award SET lease_owner=NULL,lease_until=NULL,
		next_retry_at=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL ? SECOND),retry_count=retry_count+1
		WHERE edition_id=? AND cutoff_sec=? AND user_id=? AND lease_owner=? AND delivered_at IS NULL`, 1<<min(a.Retries, 6), a.Edition, a.Cutoff, a.User, a.Owner))
}
