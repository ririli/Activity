package application

import (
	"activity/pkg/platform"
	"activity/services/activity-service/internal/appstate"
	lottery "activity/services/activity-service/internal/lottery/domain"
	"encoding/json"
	"fmt"
)

// SelectReward owns the one-time choice transition and its notification
// decision. Row locking/persistence stays behind appstate.
func SelectReward(s *appstate.State, grantID, item string) (any, error) {
	grant, err := s.LockReward(grantID)
	if err != nil {
		return nil, err
	}
	if grant.Status != "PENDING_CHOICE" {
		if grant.SelectedID == item {
			return map[string]any{"grant_id": grantID, "item_id": item, "item_name": grant.SelectedName, "quantity": grant.Quantity}, nil
		}
		return nil, platform.Fail(409, "REWARD_ALREADY_SELECTED")
	}
	var choices []lottery.Prize
	if err := json.Unmarshal(grant.Candidates, &choices); err != nil {
		return nil, err
	}
	for _, p := range choices {
		if fmt.Sprint(p.ID) != item {
			continue
		}
		if err := s.MarkRewardSelected(grantID, p); err != nil {
			return nil, err
		}
		source, id, err := s.RewardSource(grantID)
		if err != nil {
			return nil, err
		}
		p.Choices = nil
		if source == "DAY_DRAW" || source == "NIGHT_DRAW" {
			s.Notify(p, id, "normal_draw_selection")
		}
		if source == "PREVIEW" {
			s.Notify(p, id, "pay_later_selection")
		}
		return map[string]any{"grant_id": grantID, "item_id": item, "item_name": p.Name, "quantity": p.Quantity}, nil
	}
	return nil, platform.Fail(400, "INVALID_CHOICE")
}
