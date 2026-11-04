package mysql

import (
	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/domain"
	cache "activity/services/ranking-service/internal/ranking/infrastructure/redis"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"math"
	"strconv"
)

type Repository struct {
	DB    *sql.DB
	Redis *redis.Client
	Cache *cache.Cache
}

func number(s string) (int64, error) {
	n, e := strconv.ParseInt(s, 10, 64)
	if e != nil || n < 0 {
		return 0, platform.Fail(409, "INVALID_EVENT_NUMBER")
	}
	return n, nil
}
func (r *Repository) Consume(ctx context.Context, ev platform.Event) error {
	if ev.Type == "RankingAwardGranted.v1" {
		return r.receipt(ctx, ev)
	}
	var f domain.Fact
	if e := json.Unmarshal(ev.Payload, &f); e != nil {
		return platform.Fail(409, "INVALID_EVENT")
	}
	if _, e := platform.UID(f.User); e != nil {
		return platform.Fail(409, "INVALID_EVENT_USER")
	}
	if _, e := platform.UID(f.Edition); e != nil {
		return platform.Fail(409, "INVALID_EVENT_EDITION")
	}
	tx, e := r.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var seen int
	if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM inbox_event WHERE consumer_name='ranking-facts' AND event_id=?`, ev.ID).Scan(&seen); e != nil {
		return e
	}
	if seen > 0 {
		// Replayed events stay idempotent even after the settlement freezes.
		return tx.Commit()
	}
	var settlementStatus string
	e = tx.QueryRowContext(ctx, `SELECT status FROM ranking_settlement WHERE edition_id=?`, f.Edition).Scan(&settlementStatus)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return e
	}
	if settlementStatus == "FROZEN" || settlementStatus == "AWARDING" || settlementStatus == "COMPLETED" {
		return platform.Fail(409, "SETTLEMENT_FROZEN")
	}
	created, ce := number(f.Created)
	if ev.Type == "FormalDrawCommitted.v1" {
		created, ce = number(f.Occurred)
	}
	if ce != nil {
		return ce
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO ranking_entry(edition_id,user_id,score_reached_at_sec) VALUES(?,?,?) ON DUPLICATE KEY UPDATE user_id=user_id`, f.Edition, f.User, created)
	if e != nil {
		return e
	}
	var score, seq, reached int64
	e = tx.QueryRowContext(ctx, `SELECT score,last_contiguous_seq,score_reached_at_sec FROM ranking_entry WHERE edition_id=? AND user_id=? FOR UPDATE`, f.Edition, f.User).Scan(&score, &seq, &reached)
	if e != nil {
		return e
	}
	priorScore := score
	if ev.Type == "FormalDrawCommitted.v1" {
		n, e := number(f.Seq)
		if e != nil || n == 0 {
			return platform.Fail(409, "INVALID_SEQUENCE")
		}
		cost, e := number(f.Consumed)
		if e != nil || cost == 0 {
			return platform.Fail(409, "INVALID_SCORE")
		}
		if _, e = platform.UID(f.Draw); e != nil {
			return platform.Fail(409, "INVALID_DRAW")
		}
		var oldUser, oldEdition, oldSeq, oldCost, oldTime string
		e = tx.QueryRowContext(ctx, `SELECT CAST(user_id AS CHAR),CAST(edition_id AS CHAR),CAST(draw_seq AS CHAR),CAST(consumed_petals AS CHAR),CAST(occurred_at_sec AS CHAR) FROM score_fact WHERE draw_id=? OR (user_id=? AND edition_id=? AND draw_seq=?)`, f.Draw, f.User, f.Edition, n).Scan(&oldUser, &oldEdition, &oldSeq, &oldCost, &oldTime)
		if e == nil {
			if oldUser != f.User || oldEdition != f.Edition || oldSeq != f.Seq || oldCost != f.Consumed || oldTime != f.Occurred {
				return platform.Fail(409, "FACT_PAYLOAD_CONFLICT")
			}
		} else if errors.Is(e, sql.ErrNoRows) {
			_, e = tx.ExecContext(ctx, `INSERT INTO score_fact(draw_id,user_id,edition_id,draw_seq,consumed_petals,occurred_at_sec) VALUES(?,?,?,?,?,?)`, f.Draw, f.User, f.Edition, n, cost, created)
			if e != nil {
				return e
			}
		} else {
			return e
		}
		for {
			var delta, at int64
			e = tx.QueryRowContext(ctx, `SELECT consumed_petals,occurred_at_sec FROM score_fact WHERE user_id=? AND edition_id=? AND draw_seq=?`, f.User, f.Edition, seq+1).Scan(&delta, &at)
			if errors.Is(e, sql.ErrNoRows) {
				break
			}
			if e != nil {
				return e
			}
			if score > math.MaxInt64-delta {
				return platform.Fail(409, "SCORE_OVERFLOW")
			}
			if seq > 0 && at < reached {
				return platform.Fail(409, "TIME_REGRESSION")
			}
			score += delta
			seq++
			reached = at
		}
		_, e = tx.ExecContext(ctx, `UPDATE ranking_entry SET score=?,last_contiguous_seq=?,score_reached_at_sec=? WHERE edition_id=? AND user_id=?`, score, seq, reached, f.Edition, f.User)
		if e != nil {
			return e
		}
		if seq == n && f.Total != "" && f.Total != fmt.Sprint(score) {
			return platform.Fail(409, "CUMULATIVE_SCORE_MISMATCH")
		}
	} else if ev.Type != "ParticipationInitialized.v1" {
		return platform.Fail(409, "INVALID_EVENT_TYPE")
	}
	_, e = tx.ExecContext(ctx, `INSERT IGNORE INTO inbox_event(consumer_name,event_id) VALUES('ranking-facts',?)`, ev.ID)
	if e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	return r.cacheUpdate(ctx, ev, priorScore > 0)
}

type Entry = domain.Entry

func (r *Repository) Board(ctx context.Context, edition, user string) (domain.Board, error) {
	if r.Cache != nil {
		if b, e := r.Cache.Board(ctx, edition, user); e == nil {
			var users, mine int64
			if e = r.DB.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(user_id=?),0) FROM ranking_entry WHERE edition_id=?`, user, edition).Scan(&users, &mine); e == nil && b.Count >= users && (mine == 0 || b.Me != nil) {
				return domain.Board{Top: b.Top, Me: b.Me, Source: "redis"}, nil
			}
		}
	}
	tx, e := r.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return domain.Board{}, e
	}
	defer tx.Rollback()
	rows, e := tx.QueryContext(ctx, `SELECT CAST(user_id AS CHAR),score,score_reached_at_sec FROM ranking_entry WHERE edition_id=? ORDER BY score DESC,score_reached_at_sec,user_id LIMIT 20`, edition)
	if e != nil {
		return domain.Board{}, e
	}
	top := []Entry{}
	for rows.Next() {
		var v Entry
		if e = rows.Scan(&v.User, &v.Score, &v.Reached); e != nil {
			rows.Close()
			return domain.Board{}, e
		}
		v.Rank = int64(len(top) + 1)
		top = append(top, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return domain.Board{}, e
	}
	var me Entry
	me.User = user
	e = tx.QueryRowContext(ctx, `SELECT score,score_reached_at_sec FROM ranking_entry WHERE edition_id=? AND user_id=?`, edition, user).Scan(&me.Score, &me.Reached)
	var mine *Entry
	if e == nil {
		e = tx.QueryRowContext(ctx, `SELECT COUNT(*)+1 FROM ranking_entry WHERE edition_id=? AND (score>? OR(score=? AND score_reached_at_sec<?) OR(score=? AND score_reached_at_sec=? AND user_id<?))`, edition, me.Score, me.Score, me.Reached, me.Score, me.Reached, user).Scan(&me.Rank)
		if e != nil {
			return domain.Board{}, e
		}
		mine = &me
	} else if !errors.Is(e, sql.ErrNoRows) {
		return domain.Board{}, e
	}
	if e = tx.Commit(); e != nil {
		return domain.Board{}, e
	}
	return domain.Board{Top: top, Me: mine, Source: "mysql"}, nil
}
func (r *Repository) receipt(ctx context.Context, ev platform.Event) error {
	var p struct {
		Award   string `json:"award_id"`
		User    string `json:"user_id"`
		Edition string `json:"edition_id"`
	}
	if e := json.Unmarshal(ev.Payload, &p); e != nil {
		return e
	}
	res, e := r.DB.ExecContext(ctx, `UPDATE ranking_award SET delivery_status='DELIVERED',delivered_at=COALESCE(delivered_at,UTC_TIMESTAMP(6)) WHERE award_id=? AND user_id=? AND edition_id=?`, p.Award, p.User, p.Edition)
	if e != nil {
		return e
	}
	n, e := res.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		var n int
		e = r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM ranking_award WHERE award_id=? AND user_id=? AND edition_id=? AND delivery_status='DELIVERED'`, p.Award, p.User, p.Edition).Scan(&n)
		if e != nil {
			return e
		}
		if n == 0 {
			return platform.Fail(409, "UNKNOWN_AWARD_RECEIPT")
		}
	}
	return nil
}
