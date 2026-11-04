package mysql

import (
	"activity/pkg/platform"
	lottery "activity/services/activity-service/internal/lottery/domain"
	"activity/services/activity-service/internal/participation/application"
	"activity/services/activity-service/internal/participation/domain"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

func (s *state) purchase(c application.Command) (any, error) {
	var oldType, oldProduct string
	var oldPetals int64
	var oldID uint64
	e := s.tx.QueryRowContext(s.ctx, `SELECT order_id,purchase_type,product_code,petals_granted FROM purchase_order WHERE user_id=? AND request_id=?`, s.user, c.Operation).Scan(&oldID, &oldType, &oldProduct, &oldPetals)
	if e == nil {
		if oldProduct != c.Product || (oldType == "TOP_UP" && oldPetals != c.Petals) {
			return nil, platform.Fail(409, "PURCHASE_CONFLICT")
		}
		return map[string]string{"order_id": fmt.Sprint(oldID)}, nil
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return nil, e
	}
	var cost, petals int64
	limited := true
	kind := "SHOP"
	switch c.Product {
	case "5":
		cost = 300
		petals = 5
		limited = false
	case "52":
		cost = 3000
		petals = 52
	case "525":
		cost = 30000
		petals = 525
	case "1580":
		cost = 90000
		petals = 1580
	case "gift":
		kind = "SIMULATED_GIFT"
		petals = 100
	case "top-up":
		kind = "TOP_UP"
		limited = false
		petals = c.Petals
		if petals <= 0 || petals > math.MaxInt64/60 {
			return nil, platform.Fail(400, "INVALID_PETALS")
		}
		cost = petals * 60
	default:
		return nil, platform.Fail(400, "INVALID_PRODUCT")
	}
	if limited {
		var n int
		e = s.tx.QueryRowContext(s.ctx, `SELECT COUNT(*) FROM purchase_quota WHERE user_id=? AND edition_id=? AND product_code=?`, s.user, s.edition, c.Product).Scan(&n)
		if e != nil {
			return nil, e
		}
		if n > 0 {
			return nil, platform.Fail(409, "ALREADY_PURCHASED")
		}
	}
	if e = s.asset("GOLD", -cost, "PURCHASE_COST", "PURCHASE"); e != nil {
		return nil, e
	}
	if e = s.asset("PETAL", petals, "PURCHASE_GRANT", "PURCHASE"); e != nil {
		return nil, e
	}
	_, e = s.tx.ExecContext(s.ctx, `INSERT INTO purchase_order(order_id,user_id,edition_id,request_id,purchase_type,product_code,gold_cost,petals_granted,rule_version) VALUES(?,?,?,?,?,?,?,?,?)`, s.biz, s.user, s.edition, c.Operation, kind, c.Product, cost, petals, s.version)
	if e != nil {
		return nil, e
	}
	if limited {
		_, e = s.tx.ExecContext(s.ctx, `INSERT INTO purchase_quota(user_id,edition_id,product_code,order_id) VALUES(?,?,?,?)`, s.user, s.edition, c.Product, s.biz)
	}
	return map[string]any{"order_id": fmt.Sprint(s.biz), "gold": s.gold, "petals": s.petals}, e
}
func (s *state) stage(stage int, source string) (any, error) {
	p, ok := s.rules.Stages[stage]
	if !ok {
		return nil, platform.Fail(400, "INVALID_STAGE")
	}
	var grant uint64
	e := s.tx.QueryRowContext(s.ctx, `SELECT reward_grant_id FROM stage_claim WHERE user_id=? AND edition_id=? AND round_no=? AND stage_code=?`, s.user, s.edition, s.round, stage).Scan(&grant)
	if e == nil {
		return map[string]string{"grant_id": fmt.Sprint(grant)}, nil
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return nil, e
	}
	if s.flowers < stage {
		return nil, platform.Fail(409, "STAGE_NOT_REACHED")
	}
	id := platform.ID()
	grant, e = s.grant(p, "STAGE", id, 1, nil)
	if e != nil {
		return nil, e
	}
	if p.ID == 1001 {
		e = s.asset("GOLD", p.Quantity, "STAGE_REWARD", "STAGE")
	}
	if p.ID == 1002 {
		e = s.asset("PETAL", p.Quantity, "STAGE_REWARD", "STAGE")
	}
	if e != nil {
		return nil, e
	}
	_, e = s.tx.ExecContext(s.ctx, `INSERT INTO stage_claim(claim_id,user_id,edition_id,round_no,stage_code,reward_grant_id,claim_source) VALUES(?,?,?,?,?,?,?)`, id, s.user, s.edition, s.round, stage, grant, source)
	if e != nil {
		return nil, e
	}
	bit := 0
	for i, v := range []int{3, 5, 8, 11, 15, 17} {
		if v == stage {
			bit = 1 << i
		}
	}
	s.mask |= bit
	_, e = s.tx.ExecContext(s.ctx, `UPDATE user_round SET claimed_stage_mask=? WHERE user_id=? AND edition_id=? AND round_no=?`, s.mask, s.user, s.edition, s.round)
	return map[string]string{"grant_id": fmt.Sprint(grant)}, e
}
func (s *state) chestOpen() (any, error) {
	var id, grant uint64
	var raw []byte
	e := s.tx.QueryRowContext(s.ctx, `SELECT chest_id,reward_grant_id,candidate_snapshot FROM chest_open WHERE user_id=? AND edition_id=? AND round_no=? AND status='OPENED'`, s.user, s.edition, s.round).Scan(&id, &grant, &raw)
	if e == nil {
		return map[string]any{"chest_id": fmt.Sprint(id), "grant_id": fmt.Sprint(grant), "reward": json.RawMessage(raw)}, nil
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return nil, e
	}
	if s.completed >= s.unlocked {
		return nil, platform.Fail(409, "NO_CHEST_AVAILABLE")
	}
	rows, e := s.tx.QueryContext(s.ctx, `SELECT candidate_code FROM chest_open WHERE user_id=? AND edition_id=? AND round_no=?`, s.user, s.edition, s.round)
	if e != nil {
		return nil, e
	}
	used := map[string]bool{}
	for rows.Next() {
		var code string
		if e = rows.Scan(&code); e != nil {
			rows.Close()
			return nil, e
		}
		used[code] = true
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	choices := []domain.Prize{}
	for _, p := range s.rules.Chests {
		if !used[p.Code] {
			choices = append(choices, p)
		}
	}
	index, e := domain.Random(int64(len(choices)))
	if e != nil {
		return nil, e
	}
	p := choices[index]
	id = platform.ID()
	grant, e = s.grant(p, "CHEST", id, 1, nil)
	if e != nil {
		return nil, e
	}
	kind := "ITEM"
	if len(p.Choices) > 0 {
		kind = "CHOICE"
	}
	_, e = s.tx.ExecContext(s.ctx, `INSERT INTO chest_open(chest_id,user_id,edition_id,round_no,chest_no,candidate_code,reward_type,candidate_snapshot,reward_grant_id) VALUES(?,?,?,?,?,?,?,?,?)`, id, s.user, s.edition, s.round, s.completed+1, p.Code, kind, platform.JSON(p), grant)
	return map[string]any{"chest_id": fmt.Sprint(id), "grant_id": fmt.Sprint(grant), "reward": p}, e
}
func (s *state) chestConfirm(id string) (any, error) {
	var status, reward string
	var round int
	e := s.tx.QueryRowContext(s.ctx, `SELECT c.status,r.status,c.round_no FROM chest_open c JOIN reward_grant r ON c.reward_grant_id=r.grant_id WHERE c.chest_id=? AND c.user_id=? AND c.edition_id=? FOR UPDATE`, id, s.user, s.edition).Scan(&status, &reward, &round)
	if e != nil {
		return nil, e
	}
	if status == "CONFIRMED" {
		return map[string]string{"status": status}, nil
	}
	if round != s.round || reward == "PENDING_CHOICE" {
		return nil, platform.Fail(409, "CHEST_NOT_READY")
	}
	_, e = s.tx.ExecContext(s.ctx, `UPDATE chest_open SET status='CONFIRMED',confirmed_at=UTC_TIMESTAMP(6) WHERE chest_id=?`, id)
	if e != nil {
		return nil, e
	}
	_, e = s.tx.ExecContext(s.ctx, `UPDATE user_round SET completed_chests=completed_chests+1 WHERE user_id=? AND edition_id=? AND round_no=?`, s.user, s.edition, s.round)
	return map[string]string{"status": "CONFIRMED"}, e
}
func (s *state) selectReward(id, item string) (any, error) {
	var status string
	var candidates []byte
	var selected sql.NullString
	e := s.tx.QueryRowContext(s.ctx, `SELECT status,candidate_snapshot,CAST(final_item_id AS CHAR) FROM reward_grant WHERE grant_id=? AND user_id=? AND edition_id=? FOR UPDATE`, id, s.user, s.edition).Scan(&status, &candidates, &selected)
	if e != nil {
		return nil, e
	}
	if status != "PENDING_CHOICE" {
		if selected.Valid && selected.String == item {
			return map[string]string{"grant_id": id, "item_id": item}, nil
		}
		return nil, platform.Fail(409, "REWARD_ALREADY_SELECTED")
	}
	var choices []domain.Prize
	if e = json.Unmarshal(candidates, &choices); e != nil {
		return nil, e
	}
	for _, p := range choices {
		if fmt.Sprint(p.ID) == item {
			_, e = s.tx.ExecContext(s.ctx, `UPDATE reward_grant SET status='SELECTED',final_item_id=?,final_item_name=?,selected_at=UTC_TIMESTAMP(6) WHERE grant_id=? AND user_id=?`, p.ID, p.Name, id, s.user)
			if e == nil {
				var source string
				var grant uint64
				lookupErr := s.tx.QueryRowContext(s.ctx, `SELECT source_type,grant_id FROM reward_grant WHERE grant_id=? AND user_id=?`, id, s.user).Scan(&source, &grant)
				if lookupErr != nil {
					return nil, lookupErr
				}
				p.Choices = nil
				if source == "DAY_DRAW" || source == "NIGHT_DRAW" {
					s.notify(p, grant, "normal_draw_selection")
				}
				if source == "PREVIEW" {
					s.notify(p, grant, "pay_later_selection")
				}
			}
			return map[string]string{"grant_id": id, "item_id": item}, e
		}
	}
	return nil, platform.Fail(400, "INVALID_CHOICE")
}
func (s *state) nextRound(from int) (any, error) {
	if from > 0 && s.round == from+1 {
		return s.summary(), nil
	}
	if from != s.round || s.flowers != 18 || s.completed != 3 || s.mask != 63 {
		return nil, platform.Fail(409, "ROUND_NOT_COMPLETE")
	}
	var pending int
	e := s.tx.QueryRowContext(s.ctx, `SELECT COUNT(*) FROM reward_grant WHERE user_id=? AND edition_id=? AND round_no=? AND source_type='STAGE' AND status='PENDING_CHOICE'`, s.user, s.edition, s.round).Scan(&pending)
	if e != nil {
		return nil, e
	}
	if pending != 0 {
		return nil, platform.Fail(409, "STAGE_CHOICE_PENDING")
	}
	_, e = s.tx.ExecContext(s.ctx, `UPDATE user_round SET status='COMPLETED',completed_at=UTC_TIMESTAMP(6) WHERE user_id=? AND edition_id=? AND round_no=?`, s.user, s.edition, s.round)
	if e != nil {
		return nil, e
	}
	_, e = s.tx.ExecContext(s.ctx, `INSERT INTO user_round(user_id,edition_id,round_no,rule_version) VALUES(?,?,?,?)`, s.user, s.edition, s.round+1, s.version)
	if e != nil {
		return nil, e
	}
	_, e = s.tx.ExecContext(s.ctx, `UPDATE user_participation SET current_round_no=current_round_no+1,version=version+1 WHERE user_id=? AND edition_id=?`, s.user, s.edition)
	return map[string]int{"round_no": s.round + 1}, e
}
func (s *state) preview() (any, error) {
	var id uint64
	var raw []byte
	var status string
	e := s.tx.QueryRowContext(s.ctx, `SELECT preview_id,result_json,status FROM preview_order WHERE user_id=? AND edition_id=?`, s.user, s.edition).Scan(&id, &raw, &status)
	if e == nil {
		if status != "PENDING" {
			return nil, platform.Fail(409, "PREVIEW_UNAVAILABLE")
		}
		return map[string]any{"preview_id": fmt.Sprint(id), "results": json.RawMessage(raw)}, nil
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return nil, e
	}
	if s.formal != 0 {
		return nil, platform.Fail(409, "PREVIEW_UNAVAILABLE")
	}
	plan, e := domain.Simulate(s.rules.Pools["DAY"], lottery.Progress{Flowers: s.flowers, GoldValue: s.value}, 180, domain.Random)
	if e != nil {
		return nil, e
	}
	plan.Refund = 0
	id = platform.ID()
	raw = platform.JSON(plan.Steps)
	hash := canonicalHash(raw)
	_, e = s.tx.ExecContext(s.ctx, `INSERT INTO preview_order(preview_id,user_id,edition_id,round_no,rule_version,start_progress_snapshot,start_progress_version,actual_count,required_petals,result_json,expected_progress_json,result_hash) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, id, s.user, s.edition, s.round, s.version, platform.JSON(map[string]any{"flowers": s.flowers, "gold_value": s.value}), s.progress, len(plan.Steps), plan.Cost, raw, platform.JSON(plan), hash[:])
	return map[string]any{"preview_id": fmt.Sprint(id), "results": plan.Steps, "required_petals": plan.Cost}, e
}
func (s *state) confirmPreview() (any, error) {
	var id, draw uint64
	var round int
	var version int64
	var status string
	var raw, expected, hash []byte
	e := s.tx.QueryRowContext(s.ctx, `SELECT preview_id,round_no,start_progress_version,status,result_json,expected_progress_json,result_hash,COALESCE(confirmed_draw_id,0) FROM preview_order WHERE user_id=? AND edition_id=? FOR UPDATE`, s.user, s.edition).Scan(&id, &round, &version, &status, &raw, &expected, &hash, &draw)
	if e != nil {
		return nil, e
	}
	if status == "CONFIRMED" {
		return map[string]string{"draw_id": fmt.Sprint(draw)}, nil
	}
	if status != "PENDING" || s.formal != 0 || round != s.round || version != s.progress {
		return nil, platform.Fail(409, "PREVIEW_STALE")
	}
	actual := canonicalHash(raw)
	if string(actual[:]) != string(hash) {
		return nil, errors.New("preview checksum mismatch")
	}
	var plan domain.Plan
	if e = json.Unmarshal(expected, &plan); e != nil {
		return nil, e
	}
	var steps []domain.Step
	if e = json.Unmarshal(raw, &steps); e != nil {
		return nil, e
	}
	if string(platform.JSON(plan.Steps)) != string(platform.JSON(steps)) || plan.Cost <= 0 || plan.Cost != int64(len(plan.Steps))*s.rules.Pools["DAY"].Cost {
		return nil, errors.New("invalid preview plan")
	}
	gap := max(int64(0), plan.Cost-s.petals)
	if gap > 0 {
		if gap > math.MaxInt64/60 {
			return nil, errors.New("top up overflow")
		}
		if e = s.asset("GOLD", -gap*60, "PREVIEW_TOP_UP_COST", "PREVIEW"); e != nil {
			return nil, e
		}
		if e = s.asset("PETAL", gap, "PREVIEW_TOP_UP_GRANT", "PREVIEW"); e != nil {
			return nil, e
		}
		_, e = s.tx.ExecContext(s.ctx, `INSERT INTO purchase_order(order_id,user_id,edition_id,request_id,purchase_type,product_code,gold_cost,petals_granted,rule_version,parent_confirm_id) VALUES(?,?,?,?,'PREVIEW_TOP_UP','top-up',?,?,?,?)`, platform.ID(), s.user, s.edition, "preview:"+fmt.Sprint(id), gap*60, gap, s.version, id)
		if e != nil {
			return nil, e
		}
	}
	out, e := s.draw(application.Command{Edition: fmt.Sprint(s.edition), Operation: "preview:" + fmt.Sprint(id), Pool: "DAY", Count: 180}, &plan, id)
	if e != nil {
		return nil, e
	}
	_, e = s.tx.ExecContext(s.ctx, `UPDATE preview_order SET status='CONFIRMED',confirmed_draw_id=?,confirmed_at=UTC_TIMESTAMP(6) WHERE preview_id=?`, s.biz, id)
	return out, e
}
func (s *state) abandonPreview() (any, error) {
	res, e := s.tx.ExecContext(s.ctx, `UPDATE preview_order SET status='ABANDONED',abandoned_at=UTC_TIMESTAMP(6) WHERE user_id=? AND edition_id=? AND status='PENDING'`, s.user, s.edition)
	if e != nil {
		return nil, e
	}
	n, e := res.RowsAffected()
	if n == 0 {
		var status string
		e = s.tx.QueryRowContext(s.ctx, `SELECT status FROM preview_order WHERE user_id=? AND edition_id=?`, s.user, s.edition).Scan(&status)
		if e != nil {
			return nil, e
		}
		if status != "ABANDONED" {
			return nil, platform.Fail(409, "PREVIEW_UNAVAILABLE")
		}
	}
	return map[string]string{"status": "ABANDONED"}, e
}
func (r *Repository) Read(ctx context.Context, user, edition, kind, before string) (any, error) {
	u, e := platform.UID(user)
	if e != nil {
		return nil, e
	}
	if kind == "config" {
		return queryMaps(ctx, r.DB, `SELECT CAST(edition_id AS CHAR) edition_id,name,status,starts_at,ends_at,settlement_status FROM activity_edition ORDER BY starts_at DESC LIMIT 1`)
	}
	if _, e = platform.UID(edition); e != nil {
		return nil, e
	}
	if kind == "state" {
		return queryMaps(ctx, r.DB, `SELECT w.gold_balance,w.petal_balance,p.current_round_no,p.ranking_score,r.flower_count,r.consumed_gold_equivalent,r.unlocked_chests,r.completed_chests,r.claimed_stage_mask,r.status,rule.rule_json FROM wallet w LEFT JOIN user_participation p ON p.user_id=w.user_id AND p.edition_id=? LEFT JOIN user_round r ON r.user_id=p.user_id AND r.edition_id=p.edition_id AND r.round_no=p.current_round_no LEFT JOIN activity_rule_version rule ON rule.edition_id=r.edition_id AND rule.version=r.rule_version WHERE w.user_id=?`, edition, user)
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
		query = `SELECT CAST(grant_id AS CHAR) grant_id,source_type,original_item_name,quantity,status,CAST(final_item_id AS CHAR) final_item_id,final_item_name,candidate_snapshot FROM reward_grant WHERE user_id=? AND edition_id=? AND grant_id<? ORDER BY grant_id DESC LIMIT 20`
	case "summary":
		query = `SELECT CAST(final_item_id AS CHAR) item_id,final_item_name,SUM(quantity) quantity FROM reward_grant WHERE user_id=? AND edition_id=? AND status IN ('OBTAINED','SELECTED') AND final_item_id<>1002 GROUP BY final_item_id,final_item_name`
		args = args[:2]
	case "chests":
		query = `SELECT CAST(chest_id AS CHAR) chest_id,round_no,chest_no,status,candidate_snapshot,CAST(reward_grant_id AS CHAR) grant_id FROM chest_open WHERE user_id=? AND edition_id=? AND chest_id<? ORDER BY chest_id DESC LIMIT 20`
	case "preview":
		query = `SELECT CAST(preview_id AS CHAR) preview_id,status,result_json,required_petals FROM preview_order WHERE user_id=? AND edition_id=?`
		args = args[:2]
	case "purchases":
		query = `SELECT product_code,CAST(order_id AS CHAR) order_id FROM purchase_quota WHERE user_id=? AND edition_id=?`
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

func canonicalHash(raw []byte) [32]byte {
	var v any
	if e := json.Unmarshal(raw, &v); e != nil {
		panic(e)
	}
	return sha256.Sum256(platform.JSON(v))
}
