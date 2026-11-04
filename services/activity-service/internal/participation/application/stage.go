package application

import (
	"activity/pkg/platform"
	"activity/services/activity-service/internal/appstate"
	"activity/services/activity-service/internal/uow"
	"errors"
	"fmt"
)

func ClaimStage(s *appstate.State, stage int, source string) (any, error) {
	prize, ok := s.Rules.Stages[stage]
	if !ok {
		return nil, platform.Fail(400, "INVALID_STAGE")
	}
	grant, err := s.FindStageClaim(stage)
	if err == nil {
		return map[string]string{"grant_id": fmt.Sprint(grant)}, nil
	}
	if !errors.Is(err, uow.ErrNoRows) {
		return nil, err
	}
	if s.Flowers < stage {
		return nil, platform.Fail(409, "STAGE_NOT_REACHED")
	}
	claimID := platform.ID()
	grant, err = s.Grant(prize, "STAGE", claimID, 1, nil)
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
	if err = s.InsertStageClaim(claimID, stage, grant, source); err != nil {
		return nil, err
	}
	bit := 0
	for i, v := range []int{3, 5, 8, 11, 15, 17} {
		if v == stage {
			bit = 1 << i
		}
	}
	s.Mask |= bit
	if err = s.UpdateRoundMask(s.Mask); err != nil {
		return nil, err
	}
	return map[string]string{"grant_id": fmt.Sprint(grant)}, nil
}
