package application

import (
	lottery "activity/services/activity-service/internal/lottery/domain"
	"encoding/json"
	"errors"
	"fmt"
)

func settleUser(s *work) error {
	rounds, err := s.session.Participation.ListRounds()
	if err != nil {
		return err
	}
	for _, round := range rounds {
		flowers, mask, raw, err := s.session.Participation.LockRoundForSettlement(round)
		if err != nil {
			return err
		}
		s.progress.Round = round
		s.session.BindRound(round, s.progress.Progress)
		s.progress.Flowers = flowers
		s.progress.Mask = mask
		if err = json.Unmarshal(raw, &s.rules); err != nil {
			return err
		}
		for _, stage := range []int{3, 5, 8, 11, 15, 17} {
			if s.progress.Flowers >= stage {
				if _, err = claimStage(s, stage, "SETTLEMENT"); err != nil {
					return err
				}
			}
		}
	}
	pending, err := s.session.Assets.ListPendingGrants()
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
			n, err := s.session.Assets.OwnedItemCount(choice.ID)
			if err != nil {
				return err
			}
			if n == 0 {
				selected = choice
				break
			}
		}
		if _, err := selectReward(s, grant.ID, fmt.Sprint(selected.ID)); err != nil {
			return err
		}
	}
	if err = s.session.Participation.ConfirmAllChests(); err != nil {
		return err
	}
	return s.session.Participation.RecomputeCompletedChests()
}
