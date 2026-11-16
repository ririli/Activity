package mysql

import (
	"activity/pkg/platform"
	"activity/services/activity-service/internal/infrastructure/sqlkit"
	app "activity/services/activity-service/internal/lottery/application"
)

func (s *Repository) SaveDraw(w app.DrawWrite) error {
	details := make([][]any, 0, len(w.Plan.Steps))
	flowers := make([][]any, 0, len(w.Plan.Steps))
	for i, step := range w.Plan.Steps {
		no := i + 1
		id, err := s.NewID(s.Ctx)
		if err != nil {
			return err
		}
		details = append(details, []any{id, s.User, s.Edition, w.ID, no, step.Prize.Code, step.Prize.Name, step.Prize.Quantity, w.UnitCost, step.Before, step.Probability, step.After - step.Before - step.Probability, step.After, step.GoldValue})
		if step.After > step.Before {
			id, err := s.NewID(s.Ctx)
			if err != nil {
				return err
			}
			flowers = append(flowers, []any{id, s.User, s.Edition, w.Round, w.ID, no, w.Pool, step.After - step.Before, step.After})
		}
	}

	var preview any
	if w.PreviewID != 0 {
		preview = w.PreviewID
	}
	if _, e := s.Tx.Exec(s.Ctx, "INSERT INTO "+platform.Shard("draw_order", s.User)+
		`(draw_id,user_id,edition_id,round_no,operation_id,mode,preview_id,pool_code,rule_version,requested_count,actual_count,unit_petal_cost,charged_petals,refunded_petals,consumed_petals,flowers_added,reward_snapshot_json,draw_seq,occurred_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		w.ID, s.User, s.Edition, w.Round, w.Operation, w.Mode, preview, w.Pool, w.Version, w.Requested, len(w.Plan.Steps), w.UnitCost, w.Charged, w.Plan.Refund, w.Plan.Cost, w.Plan.Flowers-w.Plan.Steps[0].Before, platform.JSON(w.Plan.Steps), w.Seq, w.Occurred); e != nil {
		return e
	}
	if e := sqlkit.BatchInsert(s.Ctx, s.Tx, platform.Shard("draw_detail", s.User),
		[]string{"detail_id", "user_id", "edition_id", "draw_id", "draw_no", "original_reward_code", "original_item_name", "quantity", "petal_cost", "flowers_before", "probability_lit", "guarantee_lit", "flowers_after", "cumulative_gold_equivalent"}, details); e != nil {
		return e
	}
	if e := sqlkit.BatchInsert(s.Ctx, s.Tx, platform.Shard("flower_lit_log", s.User),
		[]string{"log_id", "user_id", "edition_id", "round_no", "draw_id", "draw_no", "pool_code", "flowers_added", "flowers_after"}, flowers); e != nil {
		return e
	}
	if w.Idempotent {
		if _, e := s.Tx.Exec(s.Ctx, `INSERT INTO idempotency_request(user_id,operation_type,operation_id,request_hash,accepted_at,recover_until,result_user_id,result_draw_id) VALUES(?,'DRAW',?,?,?,DATE_ADD(?,INTERVAL 24 HOUR),?,?)`, s.User, w.Operation, w.RequestHash[:], w.Accepted, w.Accepted, s.User, w.ID); e != nil {
			return e
		}
	}
	return nil
}
