package application

import (
	"activity/pkg/platform"
	"activity/services/activity-service/internal/appstate"
	lottery "activity/services/activity-service/internal/lottery/domain"
	"activity/services/activity-service/internal/uow"
	"encoding/json"
	"errors"
	"fmt"
)

func Preview(s *appstate.State) (any, error) {
	id, raw, status, err := s.FindPreview()
	if err == nil {
		if status != "PENDING" {
			return nil, platform.Fail(409, "PREVIEW_UNAVAILABLE")
		}
		return map[string]any{"preview_id": fmt.Sprint(id), "results": json.RawMessage(raw)}, nil
	}
	if !errors.Is(err, uow.ErrNoRows) {
		return nil, err
	}
	if s.Formal != 0 {
		return nil, platform.Fail(409, "PREVIEW_UNAVAILABLE")
	}
	plan, err := lottery.Simulate(s.Rules.Pools["DAY"], lottery.Progress{Flowers: s.Flowers, GoldValue: s.Value}, 180, lottery.Random)
	if err != nil {
		return nil, err
	}
	plan.Refund = 0
	id = platform.ID()
	raw = platform.JSON(plan.Steps)
	hash := appstate.CanonicalHash(raw)
	snapshot := platform.JSON(map[string]any{"flowers": s.Flowers, "gold_value": s.Value})
	if err = s.InsertPreview(id, s.Round, s.Version, len(plan.Steps), plan.Cost, snapshot, raw, platform.JSON(plan), hash[:]); err != nil {
		return nil, err
	}
	return map[string]any{"preview_id": fmt.Sprint(id), "results": plan.Steps, "required_petals": plan.Cost}, nil
}

func ConfirmPreview(s *appstate.State) (any, error) {
	p, err := s.LockPreview()
	if err != nil {
		return nil, err
	}
	if p.Status == "CONFIRMED" {
		return map[string]string{"draw_id": fmt.Sprint(p.DrawID)}, nil
	}
	if p.Status != "PENDING" || s.Formal != 0 || p.Round != s.Round || p.Version != s.Progress {
		return nil, platform.Fail(409, "PREVIEW_STALE")
	}
	actual := appstate.CanonicalHash(p.Raw)
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
	if string(platform.JSON(plan.Steps)) != string(platform.JSON(steps)) || plan.Cost <= 0 || plan.Cost != int64(len(plan.Steps))*s.Rules.Pools["DAY"].Cost {
		return nil, errors.New("invalid preview plan")
	}
	gap := max(int64(0), plan.Cost-s.Petals)
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
		if err = s.InsertPreviewTopUpFor(p.ID, platform.ID(), "preview:"+fmt.Sprint(p.ID), gap*60, gap, s.Version); err != nil {
			return nil, err
		}
	}
	out, err := Draw(s, appstate.Command{Edition: fmt.Sprint(s.Edition), Operation: "preview:" + fmt.Sprint(p.ID), Pool: "DAY", Count: 180}, &plan, p.ID)
	if err != nil {
		return nil, err
	}
	if err = s.ConfirmPreview(p.ID, s.Biz); err != nil {
		return nil, err
	}
	return out, nil
}

func AbandonPreview(s *appstate.State) (any, error) {
	n, err := s.AbandonPreview()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		status, err := s.PreviewStatus()
		if err != nil {
			return nil, err
		}
		if status != "ABANDONED" {
			return nil, platform.Fail(409, "PREVIEW_UNAVAILABLE")
		}
	}
	return map[string]string{"status": "ABANDONED"}, nil
}
