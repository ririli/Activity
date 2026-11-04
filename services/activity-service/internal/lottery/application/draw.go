package application

import (
	"activity/pkg/platform"
	"activity/services/activity-service/internal/appstate"
	lottery "activity/services/activity-service/internal/lottery/domain"
	"errors"
	"fmt"
	"math"
	"time"
)

// Draw executes one formal or preview-confirm draw batch. It owns the
// simulation, wallet/progress decisions and response shape; appstate persists
// the rows inside the caller's transaction.
func Draw(s *appstate.State, c appstate.Command, fixed *lottery.Plan, previewID uint64) (any, error) {
	pool, ok := s.Rules.Pools[c.Pool]
	if !ok {
		return nil, platform.Fail(400, "INVALID_POOL")
	}
	if s.Flowers >= 18 {
		return nil, platform.Fail(409, "ROUND_FULL")
	}
	if fixed == nil && (pool.Cost <= 0 || pool.Cost > math.MaxInt64/int64(c.Count) || s.Petals < pool.Cost*int64(c.Count)) {
		return nil, platform.Fail(409, "INSUFFICIENT_PETALS")
	}
	var plan lottery.Plan
	var err error
	if fixed == nil {
		plan, err = lottery.Simulate(pool, lottery.Progress{Flowers: s.Flowers, GoldValue: s.Value}, c.Count, lottery.Random)
		if err != nil {
			return nil, err
		}
	} else {
		plan = *fixed
	}
	before := s.Flowers
	mode := "NORMAL"
	source := c.Pool + "_DRAW"
	charged := plan.Cost + plan.Refund
	if previewID != 0 {
		mode = "PREVIEW"
		source = "PREVIEW"
		charged = plan.Cost
		plan.Refund = 0
	}
	if err = s.Asset("PETAL", -charged, "DRAW_CHARGE", "DRAW"); err != nil {
		return nil, err
	}
	var gold, petals int64
	for _, step := range plan.Steps {
		if step.Prize.ID == 1001 {
			if step.Prize.Quantity > math.MaxInt64-gold {
				return nil, errors.New("prize overflow")
			}
			gold += step.Prize.Quantity
		}
		if step.Prize.ID == 1002 {
			if step.Prize.Quantity > math.MaxInt64-petals {
				return nil, errors.New("prize overflow")
			}
			petals += step.Prize.Quantity
		}
	}
	if err = s.Asset("GOLD", gold, "DRAW_WIN", "DRAW"); err != nil {
		return nil, err
	}
	if err = s.Asset("PETAL", petals, "DRAW_WIN", "DRAW"); err != nil {
		return nil, err
	}
	if err = s.Asset("PETAL", plan.Refund, "DRAW_REFUND", "DRAW"); err != nil {
		return nil, err
	}
	if plan.Cost > math.MaxInt64-s.Score {
		return nil, errors.New("score overflow")
	}
	s.Seq++
	s.Score += plan.Cost
	occurred := max(s.Now.Unix(), s.Reached)

	details := make([][]any, 0, len(plan.Steps))
	flowers := make([][]any, 0, len(plan.Steps))
	grants := make([][]any, 0, len(plan.Steps))
	grantIDs := make([]string, 0, len(plan.Steps))
	for i, step := range plan.Steps {
		no := i + 1
		details = append(details, []any{platform.ID(), s.User, s.Edition, s.Biz, no, step.Prize.Code, step.Prize.Name, step.Prize.Quantity, pool.Cost, step.Before, step.Probability, step.After - step.Before - step.Probability, step.After, step.GoldValue})
		if step.After > step.Before {
			flowers = append(flowers, []any{platform.ID(), s.User, s.Edition, s.Round, s.Biz, no, c.Pool, step.After - step.Before, step.After})
		}
		grantID, grantValues := s.GrantValues(step.Prize, source, s.Biz, no, s.Biz)
		grants = append(grants, grantValues)
		grantIDs = append(grantIDs, fmt.Sprint(grantID))
		if source != "PREVIEW" {
			s.Notify(step.Prize, grantID, "normal_draw")
		}
	}
	status := "ACTIVE"
	if plan.Flowers == 18 {
		status = "FULL"
	}
	if err = s.PersistDraw(appstate.DrawWrite{
		DrawID:       s.Biz,
		Round:        s.Round,
		Version:      s.Version,
		Mode:         mode,
		PreviewID:    previewID,
		Pool:         c.Pool,
		Requested:    c.Count,
		Actual:       len(plan.Steps),
		UnitCost:     pool.Cost,
		Charged:      charged,
		Refund:       plan.Refund,
		Consumed:     plan.Cost,
		FlowersAdded: plan.Flowers - before,
		FlowersAfter: plan.Flowers,
		GoldValue:    plan.GoldValue,
		Unlocked:     plan.Flowers / 6,
		Status:       status,
		Snapshot:     platform.JSON(plan.Steps),
		Seq:          s.Seq,
		Occurred:     time.Unix(occurred, 0).UTC(),
		Operation:    c.Operation,
		RequestHash:  appstate.RequestHash(c),
		Idempotent:   previewID == 0,
		Details:      details,
		Flowers:      flowers,
		Grants:       grants,
	}); err != nil {
		return nil, err
	}
	if err = s.Reliable("FormalDrawCommitted.v1", s.User, s.Biz, map[string]string{"user_id": fmt.Sprint(s.User), "edition_id": fmt.Sprint(s.Edition), "draw_id": fmt.Sprint(s.Biz), "draw_seq": fmt.Sprint(s.Seq), "consumed_petals": fmt.Sprint(plan.Cost), "occurred_at_sec": fmt.Sprint(occurred), "total_score": fmt.Sprint(s.Score), "score_reached_at_sec": fmt.Sprint(occurred)}); err != nil {
		return nil, err
	}
	results := make([]appstate.DrawResult, 0, len(plan.Steps))
	for i, step := range plan.Steps {
		results = append(results, appstate.DrawResult{Step: step, GrantID: grantIDs[i]})
	}
	return map[string]any{"draw_id": fmt.Sprint(s.Biz), "results": results, "consumed_petals": plan.Cost, "refunded_petals": plan.Refund, "requested_count": c.Count, "actual_count": len(plan.Steps), "flowers": plan.Flowers}, nil
}
