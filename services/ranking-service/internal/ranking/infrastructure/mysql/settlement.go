package mysql

import (
	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/domain"
	"context"

	"fmt"
)

func (r *Repository) Settle(ctx context.Context, edition string) error {
	base := platform.Env("ACTIVITY_URL", "http://localhost:8082")
	var ignored any
	if e := platform.Call(ctx, base, "/internal/activity/v1/close", map[string]string{"edition_id": edition}, &ignored); e != nil {
		return e
	}
	cursor := "0"
	for {
		var users []struct {
			User    string `json:"user_id"`
			Seq     string `json:"draw_seq"`
			Total   string `json:"total_score"`
			Reached string `json:"score_reached_at_sec"`
			Created int64  `json:"participation_created_at_sec"`
		}
		if e := platform.Call(ctx, base, "/internal/activity/v1/manifest", map[string]string{"edition_id": edition, "after": cursor}, &users); e != nil {
			return e
		}
		for _, u := range users {
			f := domain.Fact{User: u.User, Edition: edition, Created: fmt.Sprint(u.Created)}
			if e := r.Consume(ctx, platform.Event{ID: fmt.Sprint(platform.ID()), Type: "ParticipationInitialized.v1", Payload: platform.JSON(f)}); e != nil {
				return e
			}
			after := "0"
			for {
				var facts []domain.Fact
				if e := platform.Call(ctx, base, "/internal/activity/v1/facts", map[string]string{"edition_id": edition, "user_id": u.User, "after": after}, &facts); e != nil {
					return e
				}
				for _, f := range facts {
					if e := r.Consume(ctx, platform.Event{ID: f.Draw, Type: "FormalDrawCommitted.v1", Payload: platform.JSON(f)}); e != nil {
						return e
					}
					after = f.Seq
				}
				if len(facts) < 100 {
					break
				}
			}
			var score, seq string
			if e := r.DB.QueryRowContext(ctx, `SELECT CAST(score AS CHAR),CAST(last_contiguous_seq AS CHAR) FROM ranking_entry WHERE edition_id=? AND user_id=?`, edition, u.User).Scan(&score, &seq); e != nil {
				return e
			}
			if score != u.Total || seq != u.Seq {
				return platform.Fail(409, "CUTOFF_RECONCILIATION_FAILED")
			}
			cursor = u.User
		}
		if len(users) < 100 {
			break
		}
	}
	tx, e := r.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.ExecContext(ctx, `INSERT INTO ranking_settlement(edition_id,cutoff_manifest,reward_rule_snapshot) VALUES(?,JSON_OBJECT('verified',true),JSON_OBJECT('version',1)) ON DUPLICATE KEY UPDATE edition_id=edition_id`, edition)
	if e != nil {
		return e
	}
	var status string
	e = tx.QueryRowContext(ctx, `SELECT status FROM ranking_settlement WHERE edition_id=? FOR UPDATE`, edition).Scan(&status)
	if e != nil {
		return e
	}
	if status == "COMPLETED" {
		if e = tx.Commit(); e != nil {
			return e
		}
		return nil
	}
	if status == "AWARDING" {
		_, e = tx.ExecContext(ctx, `UPDATE ranking_settlement SET status='COMPLETED',completed_at=UTC_TIMESTAMP(6) WHERE edition_id=? AND NOT EXISTS(SELECT 1 FROM ranking_award WHERE edition_id=? AND delivery_status='PENDING')`, edition, edition)
		if e != nil {
			return e
		}
		if e = tx.Commit(); e != nil {
			return e
		}
		if e = r.DB.QueryRowContext(ctx, `SELECT status FROM ranking_settlement WHERE edition_id=?`, edition).Scan(&status); e != nil {
			return e
		}
		if status != "COMPLETED" {
			return platform.Fail(409, "AWARDS_PENDING")
		}
		return nil
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO ranking_snapshot(edition_id,user_id,final_rank,score,score_reached_at_sec) SELECT edition_id,user_id,ROW_NUMBER() OVER(ORDER BY score DESC,score_reached_at_sec,user_id),score,score_reached_at_sec FROM ranking_entry WHERE edition_id=?`, edition)
	if e != nil {
		return e
	}
	rows, e := tx.QueryContext(ctx, `SELECT CAST(user_id AS CHAR),final_rank FROM ranking_snapshot WHERE edition_id=? AND final_rank<=20 ORDER BY final_rank`, edition)
	if e != nil {
		return e
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
			return e
		}
		winners = append(winners, w)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
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
			award := platform.ID()
			_, e = tx.ExecContext(ctx, `INSERT INTO ranking_award(award_id,edition_id,user_id,final_rank,reward_entry_no,item_id,item_name,quantity) VALUES(?,?,?,?,?,?,?,1)`, award, edition, w.user, w.rank, i+1, p.id, p.name)
			if e != nil {
				return e
			}
			u, e := platform.UID(w.user)
			if e != nil {
				return e
			}
			e = platform.Outbox(ctx, tx, "RankingAwardRequested.v1", u, award, map[string]any{"award_id": fmt.Sprint(award), "edition_id": edition, "user_id": w.user, "item_id": p.id, "item_name": p.name, "quantity": 1, "final_rank": w.rank})
			if e != nil {
				return e
			}
		}
	}
	_, e = tx.ExecContext(ctx, `UPDATE ranking_settlement SET status='AWARDING',frozen_at=UTC_TIMESTAMP(6) WHERE edition_id=?`, edition)
	if e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	return platform.Fail(409, "AWARDS_PENDING")
}

// Kept local to the owning service: no cross-service database access.
