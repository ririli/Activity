package application

import (
	"activity/pkg/platform"
	"activity/services/activity-service/internal/appstate"
)

func NextRound(s *appstate.State, from int) (any, error) {
	if from > 0 && s.Round == from+1 {
		return s.Summary(), nil
	}
	if from != s.Round || s.Flowers != 18 || s.Completed != 3 || s.Mask != 63 {
		return nil, platform.Fail(409, "ROUND_NOT_COMPLETE")
	}
	pending, err := s.PendingStageChoices(s.Round)
	if err != nil {
		return nil, err
	}
	if pending != 0 {
		return nil, platform.Fail(409, "STAGE_CHOICE_PENDING")
	}
	if err = s.CloseRound(s.Round); err != nil {
		return nil, err
	}
	if err = s.InsertRound(s.Round+1, s.Version); err != nil {
		return nil, err
	}
	if err = s.IncrementParticipationRound(); err != nil {
		return nil, err
	}
	return map[string]int{"round_no": s.Round + 1}, nil
}
