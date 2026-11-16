package application

import (
	"activity/services/activity-service/internal/fault"
	lotteryapp "activity/services/activity-service/internal/lottery/application"
	lottery "activity/services/activity-service/internal/lottery/domain"
	"encoding/json"
	"errors"
	"fmt"
)

func preview(s *work) (any, error) {
	id, raw, status, cost, err := s.session.Lottery.FindPreview()
	if err == nil {
		if status != "PENDING" {
			return nil, fault.New("PREVIEW_UNAVAILABLE")
		}
		return map[string]any{"preview_id": fmt.Sprint(id), "results": json.RawMessage(raw), "required_petals": cost}, nil
	}
	if !errors.Is(err, fault.ErrNotFound) {
		return nil, err
	}
	if s.progress.Formal != 0 {
		return nil, fault.New("PREVIEW_UNAVAILABLE")
	}
	plan, err := s.lottery.Generate(s.rules.Pools["DAY"], s.progress.Flowers, s.progress.Value, 180)
	if err != nil {
		return nil, err
	}
	plan.Refund = 0
	id, err = s.newID()
	if err != nil {
		return nil, err
	}
	raw = encode(plan.Steps)
	hash, err := lotteryapp.CanonicalHash(raw)
	if err != nil {
		return nil, err
	}
	snapshot := encode(map[string]any{"flowers": s.progress.Flowers, "gold_value": s.progress.Value})
	if err = s.session.Lottery.InsertPreview(id, s.progress.Round, s.progress.Version, len(plan.Steps), plan.Cost, snapshot, raw, encode(plan), hash[:]); err != nil {
		return nil, err
	}
	return map[string]any{"preview_id": fmt.Sprint(id), "results": plan.Steps, "required_petals": plan.Cost}, nil
}

func confirmPreview(s *work) (any, error) {
	p, err := s.session.Lottery.LockPreview()
	if err != nil {
		return nil, err
	}
	if p.Status == "CONFIRMED" {
		return s.session.Lottery.LoadDraw(p.DrawID)
	}
	if err = p.CheckConfirm(s.progress.Round, s.progress.Progress, s.progress.Formal); err != nil {
		return nil, err
	}
	actual, err := lotteryapp.CanonicalHash(p.Raw)
	if err != nil {
		return nil, err
	}
	if string(actual[:]) != string(p.Hash) {
		return nil, errors.New("preview checksum mismatch")
	}
	var plan lottery.Plan
	if err = json.Unmarshal(p.Expected, &plan); err != nil {
		return nil, err
	}
	var steps []lottery.Step
	if err = json.Unmarshal(p.Raw, &steps); err != nil {
		return nil, err
	}
	if string(encode(plan.Steps)) != string(encode(steps)) || plan.Cost <= 0 || plan.Cost != int64(len(plan.Steps))*s.rules.Pools["DAY"].Cost {
		return nil, errors.New("invalid preview plan")
	}
	gap := max(int64(0), plan.Cost-s.wallet.Petals)
	if gap > 0 {
		if gap > int64(^uint64(0)>>1)/60 {
			return nil, errors.New("top up overflow")
		}
		if err = s.Asset("GOLD", -gap*60, "PREVIEW_TOP_UP_COST", "PREVIEW"); err != nil {
			return nil, err
		}
		if err = s.Asset("PETAL", gap, "PREVIEW_TOP_UP_GRANT", "PREVIEW"); err != nil {
			return nil, err
		}
		orderID, err := s.newID()
		if err != nil {
			return nil, err
		}
		if err = s.session.Assets.InsertPreviewTopUpFor(p.ID, orderID, "preview:"+fmt.Sprint(p.ID), gap*60, gap, s.progress.Version); err != nil {
			return nil, err
		}
	}
	out, err := draw(s, lotteryapp.DrawCommand{Edition: fmt.Sprint(s.scope.Edition), Operation: "preview:" + fmt.Sprint(p.ID), Pool: "DAY", Count: 180}, &plan, p.ID)
	if err != nil {
		return nil, err
	}
	if err = s.session.Lottery.ConfirmPreview(p.ID, s.scope.Biz); err != nil {
		return nil, err
	}
	return out, nil
}

func abandonPreview(s *work) (any, error) {
	n, err := s.session.Lottery.AbandonPreview()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		status, err := s.session.Lottery.PreviewStatus()
		if err != nil {
			return nil, err
		}
		if status != "ABANDONED" {
			return nil, fault.New("PREVIEW_UNAVAILABLE")
		}
	}
	return map[string]string{"status": "ABANDONED"}, nil
}
