package mysql

import (
	"activity/pkg/hourly"
	"activity/pkg/platform"
	app "activity/services/activity-service/internal/application"
	"activity/services/activity-service/internal/fault"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

func PrepareHourlyWindows(ctx context.Context, tx *sql.Tx, edition uint64, start, end time.Time) error {
	for at := hourly.NextCutoff(start); at.Before(end); at = at.Add(time.Hour) {
		for shard := range hourly.Shards {
			if _, err := tx.ExecContext(ctx, `INSERT INTO hourly_ranking_window(edition_id,cutoff_sec,shard_no) VALUES(?,?,?)`, edition, at.Unix(), shard); err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *outbox) admit(scope app.Scope) func(time.Time) (time.Time, error) {
	return func(end time.Time) (time.Time, error) {
		var now time.Time
		if err := o.tx.QueryRowContext(o.ctx, `SELECT UTC_TIMESTAMP(6)`).Scan(&now); err != nil {
			return now, err
		}
		shard := hourly.Shard(scope.User)
		for cutoff := hourly.NextCutoff(now); cutoff.Before(end); cutoff = hourly.NextCutoff(now) {
			var sealed sql.NullTime
			if err := o.tx.QueryRowContext(o.ctx, `SELECT sealed_at FROM hourly_ranking_window
				WHERE edition_id=? AND cutoff_sec=? AND shard_no=? FOR SHARE`, scope.Edition, cutoff.Unix(), shard).Scan(&sealed); err != nil {
				return now, err
			}
			if err := o.tx.QueryRowContext(o.ctx, `SELECT UTC_TIMESTAMP(6)`).Scan(&now); err != nil {
				return now, err
			}
			if hourly.NextCutoff(now).Equal(cutoff) {
				if sealed.Valid {
					return now, fault.New("DEPENDENCY_UNAVAILABLE")
				}
				o.position = hourly.Position{Cutoff: cutoff.Unix(), Shard: shard, Lane: hourly.Lane(scope.User)}
				break
			}
			if now.Before(cutoff) {
				return now, fault.New("DEPENDENCY_UNAVAILABLE")
			}
		}
		return now, nil
	}
}

func (o *outbox) flush(scope app.Scope) error {
	if len(o.pending) == 0 {
		return nil
	}
	p := o.position
	if _, err := o.tx.ExecContext(o.ctx, `INSERT INTO hourly_ranking_sequence(edition_id,cutoff_sec,shard_no,lane_no,last_seq)
		VALUES(?,?,?,?,?) AS incoming ON DUPLICATE KEY UPDATE last_seq=hourly_ranking_sequence.last_seq+incoming.last_seq`,
		scope.Edition, p.Cutoff, p.Shard, p.Lane, len(o.pending)); err != nil {
		return err
	}
	var last int64
	if err := o.tx.QueryRowContext(o.ctx, `SELECT last_seq FROM hourly_ranking_sequence WHERE edition_id=? AND cutoff_sec=? AND shard_no=? AND lane_no=?`,
		scope.Edition, p.Cutoff, p.Shard, p.Lane).Scan(&last); err != nil {
		return err
	}
	for i, entry := range o.pending {
		p.Seq = last - int64(len(o.pending)-i-1)
		var payload map[string]any
		if err := json.Unmarshal(entry.event.Payload, &payload); err != nil {
			return err
		}
		payload["hourly"] = p
		if _, err := o.tx.ExecContext(o.ctx, `INSERT INTO outbox_event(event_id,event_type,aggregate_id,user_id,payload,
			hourly_edition_id,hourly_cutoff_sec,hourly_shard,hourly_lane,hourly_seq) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			entry.event.ID, entry.event.Type, entry.aggregate, entry.user, platform.JSON(payload),
			scope.Edition, p.Cutoff, p.Shard, p.Lane, p.Seq); err != nil {
			return err
		}
	}
	return nil
}

func (q *Queries) HourlyEditions(ctx context.Context) ([]hourly.Edition, error) {
	rows, err := q.DB.QueryContext(ctx, `SELECT CAST(edition_id AS CHAR),starts_at,ends_at FROM activity_edition WHERE hourly_ranking_enabled AND status IN ('ACTIVE','ENDED') ORDER BY starts_at,edition_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	editions := []hourly.Edition{}
	for rows.Next() {
		var e hourly.Edition
		if err := rows.Scan(&e.ID, &e.Start, &e.End); err != nil {
			return nil, err
		}
		editions = append(editions, e)
	}
	return editions, rows.Err()
}

func (q *Queries) SealHourly(ctx context.Context, w hourly.Window) (hourly.Watermark, error) {
	var marks hourly.Watermark
	tx, err := q.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return marks, err
	}
	defer tx.Rollback()
	var due bool
	if err := tx.QueryRowContext(ctx, `SELECT UTC_TIMESTAMP(6)>=FROM_UNIXTIME(?)`, w.Cutoff).Scan(&due); err != nil {
		return marks, err
	}
	if !due {
		return marks, fault.New("HOURLY_WINDOW_NOT_DUE")
	}
	for shard := range hourly.Shards {
		var sealed sql.NullTime
		err := tx.QueryRowContext(ctx, `SELECT sealed_at FROM hourly_ranking_window WHERE edition_id=? AND cutoff_sec=? AND shard_no=? FOR UPDATE`,
			w.Edition, w.Cutoff, shard).Scan(&sealed)
		if errors.Is(err, sql.ErrNoRows) {
			return marks, fault.ErrNotFound
		}
		if err != nil {
			return marks, err
		}
		if !sealed.Valid {
			if _, err := tx.ExecContext(ctx, `UPDATE hourly_ranking_window SET sealed_at=UTC_TIMESTAMP(6) WHERE edition_id=? AND cutoff_sec=? AND shard_no=?`, w.Edition, w.Cutoff, shard); err != nil {
				return marks, err
			}
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT shard_no,lane_no,last_seq FROM hourly_ranking_sequence WHERE edition_id=? AND cutoff_sec=?`, w.Edition, w.Cutoff)
	if err != nil {
		return marks, err
	}
	defer rows.Close()
	for rows.Next() {
		var shard, lane int
		var seq int64
		if err := rows.Scan(&shard, &lane, &seq); err != nil {
			return marks, err
		}
		marks[shard][lane] = seq
	}
	if err := rows.Err(); err != nil {
		return marks, err
	}
	return marks, tx.Commit()
}

func (q *Queries) ReplayHourly(ctx context.Context, r hourly.Replay) ([]app.Event, error) {
	rows, err := q.DB.QueryContext(ctx, `SELECT CAST(event_id AS CHAR),event_type,payload FROM outbox_event
		WHERE hourly_edition_id=? AND hourly_cutoff_sec=? AND hourly_shard=? AND hourly_lane=? AND hourly_seq>? AND hourly_seq<=?
		ORDER BY hourly_seq LIMIT 256`, r.Edition, r.Cutoff, r.Shard, r.Lane, r.After, r.Through)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []app.Event{}
	for rows.Next() {
		var e app.Event
		if err := rows.Scan(&e.ID, &e.Type, &e.Payload); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (q *Queries) HourlyPositions(ctx context.Context, p hourly.Positions) ([hourly.Lanes]int64, error) {
	var positions [hourly.Lanes]int64
	rows, err := q.DB.QueryContext(ctx, `SELECT lane_no,last_seq FROM hourly_ranking_sequence WHERE edition_id=? AND cutoff_sec=? AND shard_no=?`, p.Edition, p.Cutoff, p.Shard)
	if err != nil {
		return positions, err
	}
	defer rows.Close()
	for rows.Next() {
		var lane int
		var seq int64
		if err := rows.Scan(&lane, &seq); err != nil {
			return positions, err
		}
		positions[lane] = seq
	}
	return positions, rows.Err()
}

type hourlyGrantStore struct{ tx *sql.Tx }

func (s hourlyGrantStore) FindHourlyGrant(ctx context.Context, g hourly.Grant) (uint64, error) {
	var id uint64
	var edition, settlement string
	var cutoff int64
	var rank int
	err := s.tx.QueryRowContext(ctx, `SELECT grant_id,CAST(edition_id AS CHAR),cutoff_sec,final_rank,CAST(settlement_id AS CHAR)
		FROM hourly_ranking_grant WHERE user_id=? AND (settlement_id=? OR (edition_id=? AND cutoff_sec=?))`,
		g.User, g.Settlement, g.Edition, g.Cutoff).Scan(&id, &edition, &cutoff, &rank, &settlement)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fault.ErrNotFound
	}
	if err == nil && (edition != g.Edition || cutoff != g.Cutoff || rank != g.Rank || settlement != g.Settlement) {
		return 0, fault.New("HOURLY_GRANT_CONFLICT")
	}
	return id, err
}

func (s hourlyGrantStore) RecordHourlyGrant(ctx context.Context, g hourly.Grant, id uint64) error {
	var sealed sql.NullTime
	err := s.tx.QueryRowContext(ctx, `SELECT sealed_at FROM hourly_ranking_window WHERE edition_id=? AND cutoff_sec=? AND shard_no=0`,
		g.Edition, g.Cutoff).Scan(&sealed)
	if errors.Is(err, sql.ErrNoRows) || err == nil && !sealed.Valid {
		return fault.New("HOURLY_WINDOW_NOT_SEALED")
	}
	if err != nil {
		return err
	}
	_, err = s.tx.ExecContext(ctx, `INSERT INTO hourly_ranking_grant(settlement_id,user_id,edition_id,cutoff_sec,final_rank,grant_id) VALUES(?,?,?,?,?,?)`,
		g.Settlement, g.User, g.Edition, g.Cutoff, g.Rank, id)
	return err
}
