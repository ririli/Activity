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
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Repository struct {
	DB            *sql.DB
	Cache         *cache.Cache
	editions      sync.Map
	fallbackReads atomic.Int32
}

func number(s string) (int64, error) {
	n, e := strconv.ParseInt(s, 10, 64)
	if e != nil || n < 0 || strconv.FormatInt(n, 10) != s {
		return 0, platform.Fail(409, "INVALID_EVENT_NUMBER")
	}
	return n, nil
}
func settlementPhase(ctx context.Context, tx *sql.Tx, edition string) (string, error) {
	var phase string
	err := tx.QueryRowContext(ctx, `SELECT status FROM ranking_settlement WHERE edition_id=?`, edition).Scan(&phase)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return phase, err
}

// storeFact is used by delivery and cutoff reconciliation under the edition writer.
func (r *Repository) storeFact(ctx context.Context, f domain.Fact) error {
	tx, e := r.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if f.Draw != "" {
		exists, err := checkDraw(ctx, tx, f)
		if err != nil || exists {
			return err
		}
	}
	settlementStatus, e := settlementPhase(ctx, tx, f.Edition)
	if e != nil {
		return e
	}
	if settlementStatus == "FROZEN" || settlementStatus == "AWARDING" || settlementStatus == "COMPLETED" {
		if f.Draw == "" {
			var exists bool
			err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM ranking_entry WHERE edition_id=? AND user_id=?)`, f.Edition, f.User).Scan(&exists)
			if err != nil || exists {
				return err
			}
		}
		return platform.Fail(409, "SETTLEMENT_FROZEN")
	}
	if f.Draw == "" {
		_, e = tx.ExecContext(ctx, `INSERT INTO ranking_entry(edition_id,user_id,score_reached_at_sec)
			VALUES(?,?,?) AS incoming ON DUPLICATE KEY UPDATE
			score_reached_at_sec=IF(ranking_entry.score=0,LEAST(ranking_entry.score_reached_at_sec,incoming.score_reached_at_sec),ranking_entry.score_reached_at_sec)`,
			f.Edition, f.User, f.Created)
	} else {
		e = advanceScore(ctx, tx, f)
	}
	if e != nil {
		return e
	}
	return tx.Commit()
}

func checkDraw(ctx context.Context, tx *sql.Tx, f domain.Fact) (bool, error) {
	rows, err := tx.QueryContext(ctx, `SELECT CAST(draw_id AS CHAR),CAST(user_id AS CHAR),CAST(edition_id AS CHAR),
		CAST(draw_seq AS CHAR),CAST(consumed_petals AS CHAR),CAST(occurred_at_sec AS CHAR)
		FROM score_fact WHERE draw_id=? OR (edition_id=? AND user_id=? AND draw_seq=?)`,
		f.Draw, f.Edition, f.User, f.Seq)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var old domain.Fact
		if err = rows.Scan(&old.Draw, &old.User, &old.Edition, &old.Seq, &old.Consumed, &old.Occurred); err != nil {
			return false, err
		}
		if old.Draw != f.Draw || old.User != f.User || old.Edition != f.Edition || old.Seq != f.Seq ||
			old.Consumed != f.Consumed || old.Occurred != f.Occurred {
			return false, platform.Fail(409, "FACT_PAYLOAD_CONFLICT")
		}
		found = true
	}
	return found, rows.Err()
}

func advanceScore(ctx context.Context, tx *sql.Tx, f domain.Fact) error {
	_, e := tx.ExecContext(ctx, `INSERT INTO ranking_entry(edition_id,user_id,score_reached_at_sec) VALUES(?,?,?) ON DUPLICATE KEY UPDATE user_id=user_id`, f.Edition, f.User, f.Occurred)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO score_fact(draw_id,user_id,edition_id,draw_seq,consumed_petals,occurred_at_sec) VALUES(?,?,?,?,?,?)`, f.Draw, f.User, f.Edition, f.Seq, f.Consumed, f.Occurred)
	if e != nil {
		return e
	}
	var score, seq, reached int64
	e = tx.QueryRowContext(ctx, `SELECT score,last_contiguous_seq,score_reached_at_sec FROM ranking_entry WHERE edition_id=? AND user_id=? FOR UPDATE`, f.Edition, f.User).Scan(&score, &seq, &reached)
	if e != nil {
		return e
	}
	for seq < math.MaxInt64 {
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
	if f.Seq == fmt.Sprint(seq) && f.Total != "" && f.Total != fmt.Sprint(score) {
		return platform.Fail(409, "CUMULATIVE_SCORE_MISMATCH")
	}
	_, e = tx.ExecContext(ctx, `UPDATE ranking_entry SET score=?,last_contiguous_seq=?,score_reached_at_sec=? WHERE edition_id=? AND user_id=?`, score, seq, reached, f.Edition, f.User)
	return e
}

type Entry = domain.Entry

func (r *Repository) Board(ctx context.Context, edition, user string) (domain.Board, error) {
	id, err := platform.UID(edition)
	if err != nil {
		return domain.Board{}, err
	}
	edition = strconv.FormatUint(id, 10)
	if value, ok := r.editions.Load(edition); ok {
		s := value.(*liveEdition)
		if g := s.generation.Load(); g != nil {
			op, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
			b, err := r.Cache.Board(op, g, user)
			cancel()
			if err == nil && s.generation.Load() == g {
				return b, nil
			}
			if ctx.Err() == nil {
				s.generation.CompareAndSwap(g, nil)
			}
		}
	}
	if r.fallbackReads.Add(1) > 8 {
		r.fallbackReads.Add(-1)
		return domain.Board{}, platform.Fail(503, "RANKING_BUSY")
	}
	defer r.fallbackReads.Add(-1)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	b, err := r.sqlBoard(ctx, edition, user)
	if err == nil && len(b.Top) != 0 && r.Cache != nil {
		r.edition(edition)
	}
	return b, err
}

func (r *Repository) sqlBoard(ctx context.Context, edition, user string) (domain.Board, error) {
	tx, e := r.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return domain.Board{}, e
	}
	defer tx.Rollback()
	phase, e := settlementPhase(ctx, tx, edition)
	if e != nil {
		return domain.Board{}, e
	}
	final := phase == domain.SettlementAwarding || phase == domain.SettlementCompleted
	topQuery := `SELECT CAST(user_id AS CHAR),score,score_reached_at_sec FROM ranking_entry WHERE edition_id=? ORDER BY score DESC,score_reached_at_sec,user_id LIMIT 20`
	if final {
		topQuery = `SELECT CAST(user_id AS CHAR),score,score_reached_at_sec FROM ranking_snapshot WHERE edition_id=? AND final_rank<=20 ORDER BY final_rank`
	}
	rows, e := tx.QueryContext(ctx, topQuery, edition)
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
	if final {
		e = tx.QueryRowContext(ctx, `SELECT score,score_reached_at_sec,final_rank FROM ranking_snapshot WHERE edition_id=? AND user_id=?`, edition, user).Scan(&me.Score, &me.Reached, &me.Rank)
	} else {
		e = tx.QueryRowContext(ctx, `SELECT score,score_reached_at_sec FROM ranking_entry WHERE edition_id=? AND user_id=?`, edition, user).Scan(&me.Score, &me.Reached)
	}
	var mine *Entry
	if e == nil {
		if !final {
			e = tx.QueryRowContext(ctx, `SELECT COUNT(*)+1 FROM ranking_entry WHERE edition_id=? AND (score>? OR(score=? AND score_reached_at_sec<?) OR(score=? AND score_reached_at_sec=? AND user_id<?))`, edition, me.Score, me.Score, me.Reached, me.Score, me.Reached, user).Scan(&me.Rank)
			if e != nil {
				return domain.Board{}, e
			}
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
