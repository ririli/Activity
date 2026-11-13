package mysql

import (
	"activity/pkg/platform"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

func (r *Queries) Read(ctx context.Context, user, edition, kind, before, status string) (any, error) {
	u, e := platform.UID(user)
	if e != nil {
		return nil, e
	}
	if kind == "config" {
		return queryMaps(ctx, r.DB, `SELECT CAST(edition_id AS CHAR) edition_id,name,status,starts_at,ends_at,settlement_status FROM activity_edition ORDER BY starts_at DESC LIMIT 20`)
	}
	if _, e = platform.UID(edition); e != nil {
		return nil, e
	}
	if kind == "state" {
		return queryMaps(ctx, r.DB, `SELECT w.gold_balance,w.petal_balance,p.current_round_no,p.formal_draw_count,p.ranking_score,r.flower_count,r.consumed_gold_equivalent,r.unlocked_chests,r.completed_chests,r.claimed_stage_mask,r.status,rule.rule_json FROM wallet w LEFT JOIN user_participation p ON p.user_id=w.user_id AND p.edition_id=? LEFT JOIN user_round r ON r.user_id=p.user_id AND r.edition_id=p.edition_id AND r.round_no=p.current_round_no LEFT JOIN activity_rule_version rule ON rule.edition_id=r.edition_id AND rule.version=r.rule_version WHERE w.user_id=?`, edition, user)
	}
	if before == "" {
		before = "9223372036854775807"
	}
	if _, e = parseID(before); e != nil {
		return nil, platform.Fail(400, "INVALID_CURSOR")
	}
	var query string
	args := []any{user, edition, before}
	switch kind {
	case "draws":
		query = fmt.Sprintf(`SELECT CAST(draw_id AS CHAR) draw_id,pool_code,mode,requested_count,actual_count,consumed_petals,refunded_petals,flowers_added,reward_snapshot_json,occurred_at FROM %s WHERE user_id=? AND edition_id=? AND draw_id<? ORDER BY draw_id DESC LIMIT 20`, platform.Shard("draw_order", u))
	case "flowers":
		query = fmt.Sprintf(`SELECT CAST(log_id AS CHAR) log_id,CAST(draw_id AS CHAR) draw_id,draw_no,pool_code,flowers_added,flowers_after,created_at FROM %s WHERE user_id=? AND edition_id=? AND log_id<? ORDER BY log_id DESC LIMIT 20`, platform.Shard("flower_lit_log", u))
	case "rewards":
		query = `SELECT CAST(grant_id AS CHAR) grant_id,source_type,CAST(source_id AS CHAR) source_id,source_entry_no,round_no,CAST(draw_id AS CHAR) draw_id,original_reward_code,original_item_name,quantity,status,CAST(final_item_id AS CHAR) final_item_id,final_item_name,candidate_snapshot,created_at FROM reward_grant WHERE user_id=? AND edition_id=? AND grant_id<?`
		if status != "" {
			switch status {
			case "OBTAINED", "PENDING_CHOICE", "SELECTED":
				query += ` AND status=?`
				args = append(args, status)
			default:
				return nil, platform.Fail(400, "INVALID_RECORD_STATUS")
			}
		}
		query += ` ORDER BY grant_id DESC LIMIT 20`
	case "summary":
		query = `SELECT CAST(final_item_id AS CHAR) item_id,final_item_name,SUM(quantity) quantity FROM reward_grant WHERE user_id=? AND edition_id=? AND status IN ('OBTAINED','SELECTED') AND final_item_id<>1002 GROUP BY final_item_id,final_item_name`
		args = args[:2]
	case "chests":
		query = `SELECT CAST(chest_id AS CHAR) chest_id,round_no,chest_no,status,candidate_snapshot,CAST(reward_grant_id AS CHAR) grant_id FROM chest_open WHERE user_id=? AND edition_id=? AND chest_id<? ORDER BY chest_id DESC LIMIT 20`
	case "preview":
		query = `SELECT CAST(preview_id AS CHAR) preview_id,status,result_json,required_petals FROM preview_order WHERE user_id=? AND edition_id=?`
		args = args[:2]
	case "purchases":
		query = `SELECT product_code,CAST(order_id AS CHAR) order_id FROM purchase_order WHERE user_id=? AND edition_id=? ORDER BY order_id DESC`
		args = args[:2]
	default:
		return nil, platform.Fail(400, "INVALID_RECORD_TYPE")
	}
	return queryMaps(ctx, r.DB, query, args...)
}

func queryMaps(ctx context.Context, db *sql.DB, query string, args ...any) ([]map[string]any, error) {
	rows, e := db.QueryContext(ctx, query, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	cols, e := rows.Columns()
	if e != nil {
		return nil, e
	}
	out := []map[string]any{}
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if e = rows.Scan(ptrs...); e != nil {
			return nil, e
		}
		m := map[string]any{}
		for i, k := range cols {
			if b, ok := values[i].([]byte); ok {
				if json.Valid(b) && (len(b) > 0 && (b[0] == '[' || b[0] == '{')) {
					values[i] = json.RawMessage(b)
				} else {
					values[i] = string(b)
				}
			}
			m[k] = values[i]
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
