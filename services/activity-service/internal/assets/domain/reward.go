package domain

import (
	"activity/services/activity-service/internal/fault"
	"fmt"
)

type RewardGrant struct {
	Status, SelectedID, SelectedName string
	Quantity                         int64
	Candidates                       []Prize
}

func (g RewardGrant) Select(item string) (Prize, bool, error) {
	if g.Status != "PENDING_CHOICE" {
		if g.SelectedID == item {
			return Prize{Name: g.SelectedName, Quantity: g.Quantity}, false, nil
		}
		return Prize{}, false, fault.New("REWARD_ALREADY_SELECTED")
	}
	for _, p := range g.Candidates {
		if fmt.Sprint(p.ID) == item {
			p.Choices = nil
			return p, true, nil
		}
	}
	return Prize{}, false, fault.New("INVALID_CHOICE")
}
