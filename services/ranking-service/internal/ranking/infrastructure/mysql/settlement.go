package mysql

import (
	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/domain"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type settleCursor struct {
	ManifestAfter string `json:"manifest_after,omitempty"`
	CurrentUser   string `json:"current_user,omitempty"`
	FactsAfter    string `json:"facts_after,omitempty"`
	Participants  int64  `json:"participants,omitempty"`
	Facts         int64  `json:"facts,omitempty"`
}

type manifestUser struct {
	User    string `json:"user_id"`
	Seq     string `json:"draw_seq"`
	Total   string `json:"total_score"`
	Reached string `json:"score_reached_at_sec"`
	Created int64  `json:"participation_created_at_sec"`
}

func leaseOwner() string {
	owner := fmt.Sprintf("%s-%d", platform.Env("NODE_ID", "ranking"), os.Getpid())
	if len(owner) > 31 {
		owner = owner[:31]
	}
	return owner + "-" + platform.Random()
}

func activityBase() string { return platform.Env("ACTIVITY_URL", "http://localhost:8082") }

// Advance drives one bounded step of the ranking settlement state machine.
// It is safe to call repeatedly; cursors and leases make reconciliation and the
// final snapshot resumable.
func (r *Repository) Advance(ctx context.Context, edition string) (domain.SettlementResult, error) {
	id, err := platform.UID(edition)
	if err != nil {
		return domain.SettlementResult{}, err
	}
	edition = strconv.FormatUint(id, 10)
	s := r.edition(edition)
	if e := s.lock(ctx); e != nil {
		return domain.SettlementResult{}, e
	}
	defer s.unlock()
	if _, e := r.DB.ExecContext(ctx, `INSERT INTO ranking_settlement(edition_id,cutoff_manifest,reward_rule_snapshot) VALUES(?,JSON_OBJECT(),JSON_OBJECT('version',1)) ON DUPLICATE KEY UPDATE edition_id=edition_id`, edition); e != nil {
		return domain.SettlementResult{}, e
	}
	s.generation.Store(nil)
	owner := leaseOwner()
	res, e := r.DB.ExecContext(ctx, `UPDATE ranking_settlement
		SET status=IF(status='PENDING','RECONCILING',status),
		    lease_owner=?, lease_until=DATE_ADD(UTC_TIMESTAMP(6), INTERVAL 60 SECOND)
		WHERE edition_id=? AND status NOT IN ('COMPLETED','FAILED')
		  AND (lease_owner IS NULL OR lease_until<UTC_TIMESTAMP(6))
		  AND (next_retry_at IS NULL OR next_retry_at<=UTC_TIMESTAMP(6))`, owner, edition)
	if e != nil {
		return domain.SettlementResult{}, e
	}
	claimed, e := res.RowsAffected()
	if e != nil {
		return domain.SettlementResult{}, e
	}
	var status string
	var raw []byte
	if e = r.DB.QueryRowContext(ctx, `SELECT status,COALESCE(cursor_json,JSON_OBJECT()) FROM ranking_settlement WHERE edition_id=?`, edition).Scan(&status, &raw); e != nil {
		return domain.SettlementResult{}, e
	}
	if claimed == 0 {
		return r.current(ctx, edition, status)
	}
	switch status {
	case domain.SettlementCompleted:
		_ = r.releaseLease(ctx, edition, owner)
		return r.current(ctx, edition, status)
	case domain.SettlementFailed:
		_ = r.releaseLease(ctx, edition, owner)
		return domain.SettlementResult{Status: status}, platform.Fail(409, "SETTLEMENT_FAILED")
	case domain.SettlementPending, domain.SettlementReconciling:
		return r.reconcile(ctx, edition, owner, raw)
	case domain.SettlementFrozen:
		return r.freeze(ctx, edition, owner)
	case domain.SettlementAwarding:
		return r.awarding(ctx, edition, owner)
	default:
		_ = r.failSettlement(ctx, edition, owner, "UNKNOWN_SETTLEMENT_STATUS")
		return domain.SettlementResult{Status: domain.SettlementFailed}, platform.Fail(409, "SETTLEMENT_FAILED")
	}
}

func (r *Repository) ensureClosed(ctx context.Context, edition string) error {
	var ignored any
	return platform.CallTimeout(ctx, activityBase(), "/internal/activity/v1/close", map[string]string{"edition_id": edition}, &ignored, 10*time.Second)
}

func (r *Repository) reconcile(ctx context.Context, edition, owner string, raw []byte) (domain.SettlementResult, error) {
	var cur settleCursor
	if len(raw) > 0 {
		if e := json.Unmarshal(raw, &cur); e != nil {
			_ = r.failSettlement(ctx, edition, owner, "INVALID_SETTLEMENT_CURSOR")
			return domain.SettlementResult{Status: domain.SettlementFailed}, platform.Fail(409, "INVALID_SETTLEMENT_CURSOR")
		}
	}
	if e := r.ensureClosed(ctx, edition); e != nil {
		return r.retryLater(ctx, edition, owner, e)
	}
	start := time.Now()
	for {
		var users []manifestUser
		if e := platform.CallTimeout(ctx, activityBase(), "/internal/activity/v1/manifest", map[string]string{"edition_id": edition, "after": cur.ManifestAfter}, &users, 10*time.Second); e != nil {
			return r.retryLater(ctx, edition, owner, e)
		}
		if len(users) == 0 {
			break
		}
		for _, u := range users {
			if cur.CurrentUser != u.User {
				cur.CurrentUser = u.User
				cur.FactsAfter = "0"
			}
			created := fmt.Sprint(u.Created)
			hasReached := u.Reached != "" && u.Reached != "0"
			if u.Seq == "0" && hasReached {
				created = u.Reached
			}
			f := domain.Fact{User: u.User, Edition: edition, Created: created}
			if e := validateFact("ParticipationInitialized.v1", f); e != nil {
				return r.retryLater(ctx, edition, owner, e)
			}
			if e := r.storeFact(ctx, f); e != nil {
				return r.retryLater(ctx, edition, owner, e)
			}
			cur.Participants++
			for {
				var facts []domain.Fact
				if e := platform.CallTimeout(ctx, activityBase(), "/internal/activity/v1/facts", map[string]string{"edition_id": edition, "user_id": u.User, "after": cur.FactsAfter}, &facts, 10*time.Second); e != nil {
					return r.retryLater(ctx, edition, owner, e)
				}
				for _, fact := range facts {
					if fact.Edition != edition || fact.User != u.User {
						return r.retryLater(ctx, edition, owner, platform.Fail(409, "CUTOFF_FACT_SCOPE_MISMATCH"))
					}
					if e := validateFact("FormalDrawCommitted.v1", fact); e != nil {
						return r.retryLater(ctx, edition, owner, e)
					}
					if e := r.storeFact(ctx, fact); e != nil {
						return r.retryLater(ctx, edition, owner, e)
					}
					cur.Facts++
					cur.FactsAfter = fact.Seq
				}
				if e := r.saveCursor(ctx, edition, owner, cur); e != nil {
					return r.retryLater(ctx, edition, owner, e)
				}
				if len(facts) < 100 {
					break
				}
			}
			var score, seq, reached string
			if e := r.DB.QueryRowContext(ctx, `SELECT CAST(score AS CHAR),CAST(last_contiguous_seq AS CHAR),CAST(score_reached_at_sec AS CHAR) FROM ranking_entry WHERE edition_id=? AND user_id=?`, edition, u.User).Scan(&score, &seq, &reached); e != nil {
				return r.retryLater(ctx, edition, owner, e)
			}
			if score != u.Total || seq != u.Seq || (hasReached && reached != u.Reached) {
				_ = r.failSettlement(ctx, edition, owner, "CUTOFF_RECONCILIATION_FAILED")
				return domain.SettlementResult{Status: domain.SettlementFailed}, platform.Fail(409, "CUTOFF_RECONCILIATION_FAILED")
			}
			cur.ManifestAfter = u.User
			cur.CurrentUser = ""
			cur.FactsAfter = ""
			if e := r.saveCursor(ctx, edition, owner, cur); e != nil {
				return r.retryLater(ctx, edition, owner, e)
			}
			if time.Since(start) >= 5*time.Second {
				_ = r.releaseLease(ctx, edition, owner)
				return r.current(ctx, edition, domain.SettlementReconciling)
			}
		}
		if len(users) < 100 {
			break
		}
	}
	if e := r.setFrozen(ctx, edition, owner, cur); e != nil {
		return r.retryLater(ctx, edition, owner, e)
	}
	return r.current(ctx, edition, domain.SettlementFrozen)
}

func (r *Repository) saveCursor(ctx context.Context, edition, owner string, cur settleCursor) error {
	raw, e := json.Marshal(cur)
	if e != nil {
		return e
	}
	res, e := r.DB.ExecContext(ctx, `UPDATE ranking_settlement
		SET cursor_json=?, lease_until=DATE_ADD(UTC_TIMESTAMP(6), INTERVAL 60 SECOND)
		WHERE edition_id=? AND lease_owner=? AND status='RECONCILING'`, raw, edition, owner)
	if e != nil {
		return e
	}
	n, e := res.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return errors.New("settlement lease lost")
	}
	return nil
}

func (r *Repository) setFrozen(ctx context.Context, edition, owner string, cur settleCursor) error {
	manifest, _ := json.Marshal(map[string]any{"verified": true, "participants": cur.Participants, "facts": cur.Facts})
	res, e := r.DB.ExecContext(ctx, `UPDATE ranking_settlement
		SET status='FROZEN', frozen_at=UTC_TIMESTAMP(6), cutoff_manifest=?,
		    cursor_json=NULL, lease_owner=NULL, lease_until=NULL
		WHERE edition_id=? AND lease_owner=? AND status='RECONCILING'`, manifest, edition, owner)
	if e != nil {
		return e
	}
	n, e := res.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return errors.New("settlement lease lost")
	}
	return nil
}

func (r *Repository) freeze(ctx context.Context, edition, owner string) (domain.SettlementResult, error) {
	tx, e := r.DB.BeginTx(ctx, nil)
	if e != nil {
		return r.retryLater(ctx, edition, owner, e)
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, `INSERT IGNORE INTO ranking_snapshot(edition_id,user_id,final_rank,score,score_reached_at_sec)
		SELECT edition_id,user_id,ROW_NUMBER() OVER(ORDER BY score DESC,score_reached_at_sec,user_id),score,score_reached_at_sec
		FROM ranking_entry WHERE edition_id=?`, edition); e != nil {
		return r.retryLater(ctx, edition, owner, e)
	}
	rows, e := tx.QueryContext(ctx, `SELECT CAST(user_id AS CHAR),final_rank FROM ranking_snapshot WHERE edition_id=? AND final_rank<=20 ORDER BY final_rank`, edition)
	if e != nil {
		return r.retryLater(ctx, edition, owner, e)
	}
	type winner struct {
		user string
		rank int
	}
	winners := []winner{}
	for rows.Next() {
		var w winner
		if e = rows.Scan(&w.user, &w.rank); e != nil {
			rows.Close()
			return r.retryLater(ctx, edition, owner, e)
		}
		winners = append(winners, w)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return r.retryLater(ctx, edition, owner, e)
	}
	type awardRow struct {
		id    uint64
		user  string
		rank  int
		entry int
		item  string
		name  string
	}
	awards := []awardRow{}
	for _, w := range winners {
		items := []struct {
			id   string
			name string
		}{{"3633", "春日公主称号"}}
		if w.rank == 1 {
			items = []struct {
				id   string
				name string
			}{{"1323", "定制称号/修改V8称号"}, {"1207811", "春日公主冠名头像框"}}
		} else if w.rank <= 10 {
			items = append(items, struct {
				id   string
				name string
			}{"1207809", "春日公主头像框"})
		}
		for i, p := range items {
			awards = append(awards, awardRow{id: platform.ID(), user: w.user, rank: w.rank, entry: i + 1, item: p.id, name: p.name})
		}
	}
	for start := 0; start < len(awards); start += 100 {
		end := min(len(awards), start+100)
		values := make([]string, 0, end-start)
		args := make([]any, 0, (end-start)*8)
		for _, a := range awards[start:end] {
			values = append(values, "(?,?,?,?,?,?,?,1)")
			args = append(args, a.id, edition, a.user, a.rank, a.entry, a.item, a.name)
		}
		if _, e = tx.ExecContext(ctx, `INSERT IGNORE INTO ranking_award(award_id,edition_id,user_id,final_rank,reward_entry_no,item_id,item_name,quantity) VALUES `+joinValues(values), args...); e != nil {
			return r.retryLater(ctx, edition, owner, e)
		}
	}
	for start := 0; start < len(awards); start += 100 {
		end := min(len(awards), start+100)
		values := make([]string, 0, end-start)
		args := make([]any, 0, (end-start)*5)
		for _, a := range awards[start:end] {
			u, e := platform.UID(a.user)
			if e != nil {
				return r.retryLater(ctx, edition, owner, e)
			}
			values = append(values, "(?,?,?,?,?)")
			args = append(args, a.id, "RankingAwardRequested.v1", a.id, u, platform.JSON(map[string]any{"award_id": fmt.Sprint(a.id), "edition_id": edition, "user_id": a.user, "item_id": a.item, "item_name": a.name, "quantity": 1, "final_rank": a.rank}))
		}
		if _, e = tx.ExecContext(ctx, `INSERT IGNORE INTO outbox_event(event_id,event_type,aggregate_id,user_id,payload) VALUES `+joinValues(values), args...); e != nil {
			return r.retryLater(ctx, edition, owner, e)
		}
	}
	res, e := tx.ExecContext(ctx, `UPDATE ranking_settlement
		SET status='AWARDING', frozen_at=COALESCE(frozen_at,UTC_TIMESTAMP(6)), lease_owner=NULL, lease_until=NULL
		WHERE edition_id=? AND lease_owner=? AND status='FROZEN'`, edition, owner)
	if e != nil {
		return r.retryLater(ctx, edition, owner, e)
	}
	n, e := res.RowsAffected()
	if e != nil {
		return r.retryLater(ctx, edition, owner, e)
	}
	if n == 0 {
		return r.retryLater(ctx, edition, owner, errors.New("settlement lease lost"))
	}
	if e = tx.Commit(); e != nil {
		return r.retryLater(ctx, edition, owner, e)
	}
	return r.current(ctx, edition, domain.SettlementAwarding)
}

func joinValues(values []string) string {
	out := ""
	for i, v := range values {
		if i > 0 {
			out += ","
		}
		out += v
	}
	return out
}

func (r *Repository) awarding(ctx context.Context, edition, owner string) (domain.SettlementResult, error) {
	pending, e := r.pendingAwards(ctx, edition)
	if e != nil {
		return r.retryLater(ctx, edition, owner, e)
	}
	if pending == 0 {
		res, e := r.DB.ExecContext(ctx, `UPDATE ranking_settlement
			SET status='COMPLETED', completed_at=UTC_TIMESTAMP(6), lease_owner=NULL, lease_until=NULL
			WHERE edition_id=? AND lease_owner=? AND status='AWARDING'`, edition, owner)
		if e != nil {
			return r.retryLater(ctx, edition, owner, e)
		}
		if n, e := res.RowsAffected(); e != nil || n == 0 {
			return r.current(ctx, edition, domain.SettlementAwarding)
		}
		return r.current(ctx, edition, domain.SettlementCompleted)
	}
	_ = r.releaseLease(ctx, edition, owner)
	return domain.SettlementResult{Status: domain.SettlementAwarding, PendingAwards: pending}, nil
}

func (r *Repository) current(ctx context.Context, edition, status string) (domain.SettlementResult, error) {
	pending, e := r.pendingAwards(ctx, edition)
	if e != nil {
		return domain.SettlementResult{}, e
	}
	return domain.SettlementResult{Status: status, PendingAwards: pending}, nil
}

func (r *Repository) pendingAwards(ctx context.Context, edition string) (int64, error) {
	var n int64
	e := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM ranking_award WHERE edition_id=? AND delivery_status='PENDING'`, edition).Scan(&n)
	return n, e
}

func (r *Repository) releaseLease(ctx context.Context, edition, owner string) error {
	_, e := r.DB.ExecContext(ctx, `UPDATE ranking_settlement SET lease_owner=NULL, lease_until=NULL WHERE edition_id=? AND lease_owner=?`, edition, owner)
	return e
}

func (r *Repository) retryLater(ctx context.Context, edition, owner string, cause error) (domain.SettlementResult, error) {
	var retries int
	_ = r.DB.QueryRowContext(ctx, `SELECT retry_count FROM ranking_settlement WHERE edition_id=?`, edition).Scan(&retries)
	delay := 5 * time.Second << min(retries, 6)
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}
	msg := cause.Error()
	if len(msg) > 1024 {
		msg = msg[:1024]
	}
	_, _ = r.DB.ExecContext(ctx, `UPDATE ranking_settlement
		SET retry_count=retry_count+1, next_retry_at=?, last_error=?, lease_owner=NULL, lease_until=NULL
		WHERE edition_id=? AND lease_owner=?`, time.Now().UTC().Add(delay), msg, edition, owner)
	return domain.SettlementResult{}, cause
}

func (r *Repository) failSettlement(ctx context.Context, edition, owner, code string) error {
	_, e := r.DB.ExecContext(ctx, `UPDATE ranking_settlement
		SET status='FAILED', last_error=?, lease_owner=NULL, lease_until=NULL
		WHERE edition_id=? AND lease_owner=?`, code, edition, owner)
	return e
}
