package application

import (
	"activity/services/activity-service/internal/fault"
	"errors"
	"fmt"
)

func claimStage(s *work, stage int, source string) (any, error) {
	prize, ok := s.rules.Stages[stage]
	if !ok {
		return nil, fault.New("INVALID_STAGE")
	}
	grant, err := s.session.Participation.FindStageClaim(stage)
	if err == nil {
		return map[string]string{"grant_id": fmt.Sprint(grant)}, nil
	}
	if !errors.Is(err, fault.ErrNotFound) {
		return nil, err
	}
	if err = s.progress.Claim(stage); err != nil {
		return nil, err
	}
	claimID := s.runtime.NewID()
	grant, err = s.Grant(prize, "STAGE", claimID, 1)
	if err != nil {
		return nil, err
	}
	if prize.ID == 1001 {
		err = s.Asset("GOLD", prize.Quantity, "STAGE_REWARD", "STAGE")
	}
	if prize.ID == 1002 {
		err = s.Asset("PETAL", prize.Quantity, "STAGE_REWARD", "STAGE")
	}
	if err != nil {
		return nil, err
	}
	if err = s.session.Participation.InsertStageClaim(claimID, stage, grant, source); err != nil {
		return nil, err
	}
	if err = s.session.Participation.UpdateRoundMask(s.progress.Mask); err != nil {
		return nil, err
	}
	return map[string]string{"grant_id": fmt.Sprint(grant)}, nil
}
