package application

import (
	"activity/services/activity-service/internal/appstate"
	assetsapp "activity/services/activity-service/internal/assets/application"
	lottery "activity/services/activity-service/internal/lottery/domain"
	"encoding/json"
	"errors"
	"fmt"
)

func SettleUser(s *appstate.State) error {
	rounds, err := s.ListRounds()
	if err != nil {
		return err
	}
	for _, round := range rounds {
		flowers, mask, raw, err := s.LockRoundForSettlement(round)
		if err != nil {
			return err
		}
		s.Round = round
		s.Flowers = flowers
		s.Mask = mask
		if err = json.Unmarshal(raw, &s.Rules); err != nil {
			return err
		}
		for _, stage := range []int{3, 5, 8, 11, 15, 17} {
			if s.Flowers >= stage {
				if _, err = ClaimStage(s, stage, "SETTLEMENT"); err != nil {
					return err
				}
			}
		}
	}
	pending, err := s.ListPendingGrants()
	if err != nil {
		return err
	}
	for _, grant := range pending {
		var choices []lottery.Prize
		if err := json.Unmarshal(grant.Candidates, &choices); err != nil {
			return err
		}
		if len(choices) == 0 {
			return errors.New("empty choice")
		}
		selected := choices[0]
		for _, choice := range choices {
			n, err := s.OwnedItemCount(choice.ID)
			if err != nil {
				return err
			}
			if n == 0 {
				selected = choice
				break
			}
		}
		if _, err := assetsapp.SelectReward(s, grant.ID, fmt.Sprint(selected.ID)); err != nil {
			return err
		}
	}
	if err = s.ConfirmAllChests(); err != nil {
		return err
	}
	return s.RecomputeCompletedChests()
}
