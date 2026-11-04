package mysql

import (
	"activity/pkg/platform"
	"activity/services/activity-service/internal/participation/application"
	"activity/services/activity-service/internal/participation/domain"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func (r *Repository) Bootstrap(ctx context.Context) error {
	rows, e := r.DB.QueryContext(ctx, `SELECT CAST(user_id AS CHAR) FROM wallet ORDER BY user_id`)
	if e != nil {
		return e
	}
	users := []string{}
	for rows.Next() {
		var u string
		if e = rows.Scan(&u); e != nil {
			rows.Close()
			return e
		}
		users = append(users, u)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	rows, e = r.DB.QueryContext(ctx, `SELECT CAST(edition_id AS CHAR) FROM activity_edition WHERE status='ACTIVE' AND starts_at<=UTC_TIMESTAMP(6) AND ends_at>UTC_TIMESTAMP(6)`)
	if e != nil {
		return e
	}
	editions := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return e
		}
		editions = append(editions, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, u := range users {
		for _, edition := range editions {
			var n int
			if e = r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_participation WHERE user_id=? AND edition_id=?`, u, edition).Scan(&n); e != nil {
				return e
			}
			if n == 0 {
				if _, e = r.Execute(ctx, u, application.Command{Action: "join", Edition: edition}); e != nil {
					return e
				}
			}
		}
	}
	return nil
}
func (r *Repository) Internal(ctx context.Context, action, edition, user, after string) (any, error) {
	if _, e := platform.UID(edition); e != nil {
		return nil, e
	}
	switch action {
	case "close":
		tx, e := r.DB.BeginTx(ctx, nil)
		if e != nil {
			return nil, e
		}
		defer tx.Rollback()
		var end, now time.Time
		e = tx.QueryRowContext(ctx, `SELECT ends_at,UTC_TIMESTAMP(6) FROM activity_edition WHERE edition_id=? FOR UPDATE`, edition).Scan(&end, &now)
		if e != nil {
			return nil, e
		}
		if now.Before(end) {
			return nil, platform.Fail(409, "ACTIVITY_NOT_ENDED")
		}
		_, e = tx.ExecContext(ctx, `UPDATE activity_edition SET status='ENDED' WHERE edition_id=?`, edition)
		if e != nil {
			return nil, e
		}
		return map[string]string{"status": "ENDED"}, tx.Commit()
	case "manifest":
		if after == "" {
			after = "0"
		}
		return queryMaps(ctx, r.DB, `SELECT CAST(user_id AS CHAR) user_id,CAST(draw_seq AS CHAR) draw_seq,CAST(ranking_score AS CHAR) total_score,CAST(ranking_score_reached_at_sec AS CHAR) score_reached_at_sec,CAST(FLOOR(UNIX_TIMESTAMP(created_at)) AS UNSIGNED) participation_created_at_sec FROM user_participation WHERE edition_id=? AND user_id>? ORDER BY user_id LIMIT 100`, edition, after)
	case "facts":
		u, e := platform.UID(user)
		if e != nil {
			return nil, e
		}
		if after == "" {
			after = "0"
		}
		return queryMaps(ctx, r.DB, fmt.Sprintf(`SELECT CAST(user_id AS CHAR) user_id,CAST(edition_id AS CHAR) edition_id,CAST(draw_id AS CHAR) draw_id,CAST(draw_seq AS CHAR) draw_seq,CAST(consumed_petals AS CHAR) consumed_petals,CAST(FLOOR(UNIX_TIMESTAMP(occurred_at)) AS CHAR) occurred_at_sec FROM %s WHERE user_id=? AND edition_id=? AND draw_seq>? ORDER BY draw_seq LIMIT 100`, platform.Shard("draw_order", u)), user, edition, after)
	}
	return nil, platform.Fail(400, "INVALID_INTERNAL_ACTION")
}
func (r *Repository) GrantAward(ctx context.Context, ev platform.Event) error {
	var p struct {
		Award    string `json:"award_id"`
		User     string `json:"user_id"`
		Edition  string `json:"edition_id"`
		Item     string `json:"item_id"`
		Name     string `json:"item_name"`
		Quantity int64  `json:"quantity"`
	}
	if e := json.Unmarshal(ev.Payload, &p); e != nil {
		return platform.Fail(409, "INVALID_AWARD")
	}
	u, e := platform.UID(p.User)
	if e != nil {
		return platform.Fail(409, "INVALID_AWARD")
	}
	ed, e := platform.UID(p.Edition)
	if e != nil {
		return platform.Fail(409, "INVALID_AWARD")
	}
	award, e := platform.UID(p.Award)
	if e != nil {
		return platform.Fail(409, "INVALID_AWARD")
	}
	item, e := platform.UID(p.Item)
	if e != nil || p.Quantity <= 0 || item == 1001 || item == 1002 {
		return platform.Fail(409, "INVALID_AWARD")
	}
	tx, e := r.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var grant uint64
	e = tx.QueryRowContext(ctx, `SELECT grant_id FROM reward_grant WHERE user_id=? AND source_type='RANKING' AND source_id=?`, u, award).Scan(&grant)
	if e == sql.ErrNoRows {
		var wallet uint64
		if e = tx.QueryRowContext(ctx, `SELECT user_id FROM wallet WHERE user_id=? FOR UPDATE`, u).Scan(&wallet); e != nil {
			return e
		}
		grant = platform.ID()
		_, e = tx.ExecContext(ctx, `INSERT INTO reward_grant(grant_id,user_id,edition_id,source_type,source_id,source_entry_no,original_reward_code,original_item_name,quantity,status,final_item_id,final_item_name) VALUES(?,?,?,'RANKING',?,1,?,?,?,'OBTAINED',?,?)`, grant, u, ed, award, p.Item, p.Name, p.Quantity, item, p.Name)
		if e != nil {
			return e
		}
	} else if e != nil {
		return e
	}
	e = platform.Outbox(ctx, tx, "RankingAwardGranted.v1", u, award, map[string]string{"award_id": p.Award, "user_id": p.User, "edition_id": p.Edition, "grant_id": fmt.Sprint(grant)})
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (s *state) settleUser() error {
	rows, e := s.tx.QueryContext(s.ctx, `SELECT round_no FROM user_round WHERE user_id=? AND edition_id=? ORDER BY round_no`, s.user, s.edition)
	if e != nil {
		return e
	}
	rounds := []int{}
	for rows.Next() {
		var n int
		if e = rows.Scan(&n); e != nil {
			rows.Close()
			return e
		}
		rounds = append(rounds, n)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, round := range rounds {
		s.round = round
		var raw []byte
		e = s.tx.QueryRowContext(s.ctx, `SELECT r.flower_count,r.claimed_stage_mask,v.rule_json FROM user_round r JOIN activity_rule_version v ON v.edition_id=r.edition_id AND v.version=r.rule_version WHERE r.user_id=? AND r.edition_id=? AND r.round_no=? FOR UPDATE`, s.user, s.edition, round).Scan(&s.flowers, &s.mask, &raw)
		if e != nil {
			return e
		}
		if e = json.Unmarshal(raw, &s.rules); e != nil {
			return e
		}
		for _, stage := range []int{3, 5, 8, 11, 15, 17} {
			if s.flowers >= stage {
				if _, e = s.stage(stage, "SETTLEMENT"); e != nil {
					return e
				}
			}
		}
	}
	rows, e = s.tx.QueryContext(s.ctx, `SELECT CAST(grant_id AS CHAR),candidate_snapshot FROM reward_grant WHERE user_id=? AND edition_id=? AND status='PENDING_CHOICE' ORDER BY grant_id FOR UPDATE`, s.user, s.edition)
	if e != nil {
		return e
	}
	type pending struct {
		id      string
		choices []domain.Prize
	}
	list := []pending{}
	for rows.Next() {
		var id string
		var raw []byte
		if e = rows.Scan(&id, &raw); e != nil {
			rows.Close()
			return e
		}
		v := pending{id: id}
		if e = json.Unmarshal(raw, &v.choices); e != nil {
			rows.Close()
			return e
		}
		list = append(list, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, v := range list {
		if len(v.choices) == 0 {
			return fmt.Errorf("empty choice")
		}
		selected := v.choices[0]
		for _, p := range v.choices {
			var n int
			e = s.tx.QueryRowContext(s.ctx, `SELECT COUNT(*) FROM reward_grant WHERE user_id=? AND final_item_id=? AND status IN ('OBTAINED','SELECTED')`, s.user, p.ID).Scan(&n)
			if e != nil {
				return e
			}
			if n == 0 {
				selected = p
				break
			}
		}
		if _, e = s.selectReward(v.id, fmt.Sprint(selected.ID)); e != nil {
			return e
		}
	}
	_, e = s.tx.ExecContext(s.ctx, `UPDATE chest_open SET status='CONFIRMED',confirmed_at=UTC_TIMESTAMP(6) WHERE user_id=? AND edition_id=? AND status='OPENED'`, s.user, s.edition)
	if e != nil {
		return e
	}
	_, e = s.tx.ExecContext(s.ctx, `UPDATE user_round r SET completed_chests=(SELECT COUNT(*) FROM chest_open c WHERE c.user_id=r.user_id AND c.edition_id=r.edition_id AND c.round_no=r.round_no AND c.status='CONFIRMED') WHERE r.user_id=? AND r.edition_id=?`, s.user, s.edition)
	return e
}
func (r *Repository) Maintenance(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			op, cancel := context.WithTimeout(ctx, 30*time.Second)
			_ = r.Bootstrap(op)
			_ = r.settleEnded(op)
			cancel()
		}
	}
}
func (r *Repository) settleEnded(ctx context.Context) error {
	rows, e := r.DB.QueryContext(ctx, `SELECT CAST(edition_id AS CHAR) FROM activity_edition WHERE ends_at<=UTC_TIMESTAMP(6) AND settlement_status<>'COMPLETED'`)
	if e != nil {
		return e
	}
	editions := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return e
		}
		editions = append(editions, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, id := range editions {
		if _, e = r.Internal(ctx, "close", id, "", ""); e != nil {
			return e
		}
		users, e := queryMaps(ctx, r.DB, `SELECT CAST(user_id AS CHAR) user_id FROM user_participation WHERE edition_id=? ORDER BY user_id`, id)
		if e != nil {
			return e
		}
		for _, u := range users {
			if _, e = r.Execute(ctx, u["user_id"].(string), application.Command{Action: "settle", Edition: id}); e != nil {
				return e
			}
		}
		var out any
		e = platform.Call(ctx, platform.Env("RANKING_URL", "http://localhost:8083"), "/internal/ranking/v1/settle", map[string]string{"edition_id": id}, &out)
		if e != nil {
			return e
		}
		_, e = r.DB.ExecContext(ctx, `UPDATE activity_edition SET settlement_status='COMPLETED' WHERE edition_id=?`, id)
		if e != nil {
			return e
		}
	}
	return nil
}
