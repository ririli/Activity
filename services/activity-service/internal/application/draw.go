package application

import (
	assetsapp "activity/services/activity-service/internal/assets/application"
	"activity/services/activity-service/internal/fault"
	lotteryapp "activity/services/activity-service/internal/lottery/application"
	lottery "activity/services/activity-service/internal/lottery/domain"
	"errors"
	"fmt"
	"math"
	"time"
)

func draw(s *work, c lotteryapp.DrawCommand, fixed *lottery.Plan, previewID uint64) (any, error) {
	pool, ok := s.rules.Pools[c.Pool]
	if !ok {
		return nil, fault.New("INVALID_POOL")
	}
	if s.progress.Flowers >= 18 {
		return nil, fault.New("ROUND_FULL")
	}
	if fixed == nil && (c.Count <= 0 || pool.Cost <= 0 || pool.Cost > math.MaxInt64/int64(c.Count) || s.wallet.Petals < pool.Cost*int64(c.Count)) {
		return nil, fault.New("INSUFFICIENT_PETALS")
	}
	var plan lottery.Plan
	var err error
	if fixed == nil {
		plan, err = s.lottery.Generate(pool, s.progress.Flowers, s.progress.Value, c.Count)
		if err != nil {
			return nil, err
		}
	} else {
		plan = *fixed
	}
	mode, source := "NORMAL", c.Pool+"_DRAW"
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
	grants := make([]assetsapp.Grant, 0, len(plan.Steps))
	results := make([]lotteryapp.DrawStep, 0, len(plan.Steps))
	for i, step := range plan.Steps {
		id := s.runtime.NewID()
		grants = append(grants, assetsapp.Grant{ID: id, Round: s.progress.Round, Prize: assetPrize(step.Prize), Source: source, SourceID: s.scope.Biz, No: i + 1, DrawID: s.scope.Biz})
		results = append(results, lotteryapp.DrawStep{Step: step, GrantID: fmt.Sprint(id)})
		if source != "PREVIEW" {
			s.Notify(step.Prize, id, "normal_draw")
		}
	}
	if err = s.session.Assets.SaveGrants(grants); err != nil {
		return nil, err
	}
	if err = s.participation.ApplyDraw(&s.progress, plan.Flowers, plan.GoldValue, plan.Cost, s.now.Unix()); err != nil {
		return nil, err
	}
	occurred := s.progress.Reached
	if err = s.session.Lottery.SaveDraw(lotteryapp.DrawWrite{ID: s.scope.Biz, Round: s.progress.Round, Version: s.progress.Version, Mode: mode, PreviewID: previewID, Pool: c.Pool, Requested: c.Count, UnitCost: pool.Cost, Charged: charged, Plan: plan, Seq: s.progress.Seq, Occurred: time.Unix(occurred, 0).UTC(), Accepted: s.now, Operation: c.Operation, RequestHash: lotteryapp.RequestHash(lotteryapp.DrawCommand{Edition: c.Edition, Operation: c.Operation, Pool: c.Pool, Count: c.Count}), Idempotent: previewID == 0}); err != nil {
		return nil, err
	}
	if err = s.Reliable("FormalDrawCommitted.v1", s.scope.User, s.scope.Biz, map[string]string{"user_id": fmt.Sprint(s.scope.User), "edition_id": fmt.Sprint(s.scope.Edition), "draw_id": fmt.Sprint(s.scope.Biz), "draw_seq": fmt.Sprint(s.progress.Seq), "consumed_petals": fmt.Sprint(plan.Cost), "occurred_at_sec": fmt.Sprint(occurred), "total_score": fmt.Sprint(s.progress.Score), "score_reached_at_sec": fmt.Sprint(occurred)}); err != nil {
		return nil, err
	}
	return lotteryapp.DrawResult{DrawID: fmt.Sprint(s.scope.Biz), Results: results, Consumed: plan.Cost, Refunded: plan.Refund, Requested: c.Count, Actual: len(plan.Steps), Flowers: plan.Flowers}, nil
}
