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

func OpenChest(s *appstate.State) (any, error) {
	id, grant, raw, err := s.FindOpenChest()
	if err == nil {
		return map[string]any{"chest_id": fmt.Sprint(id), "grant_id": fmt.Sprint(grant), "reward": json.RawMessage(raw)}, nil
	}
	if !errors.Is(err, uow.ErrNoRows) {
		return nil, err
	}
	if s.Completed >= s.Unlocked {
		return nil, platform.Fail(409, "NO_CHEST_AVAILABLE")
	}
	used, err := s.UsedChestCodes()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, code := range used {
		seen[code] = true
	}
	choices := []lottery.Prize{}
	for _, p := range s.Rules.Chests {
		if !seen[p.Code] {
			choices = append(choices, p)
		}
	}
	if len(choices) == 0 {
		return nil, platform.Fail(409, "NO_CHEST_AVAILABLE")
	}
	index, err := lottery.Random(int64(len(choices)))
	if err != nil {
		return nil, err
	}
	p := choices[index]
	id = platform.ID()
	grant, err = s.Grant(p, "CHEST", id, 1, nil)
	if err != nil {
		return nil, err
	}
	kind := "ITEM"
	if len(p.Choices) > 0 {
		kind = "CHOICE"
	}
	if err = s.InsertChest(id, s.Completed+1, p.Code, kind, platform.JSON(p), grant); err != nil {
		return nil, err
	}
	return map[string]any{"chest_id": fmt.Sprint(id), "grant_id": fmt.Sprint(grant), "reward": p}, nil
}

func ConfirmChest(s *appstate.State, id string) (any, error) {
	status, reward, round, err := s.LockChest(id)
	if err != nil {
		return nil, err
	}
	if status == "CONFIRMED" {
		return map[string]string{"status": status}, nil
	}
	if round != s.Round || reward == "PENDING_CHOICE" {
		return nil, platform.Fail(409, "CHEST_NOT_READY")
	}
	if err = s.ConfirmChest(id); err != nil {
		return nil, err
	}
	if err = s.IncrementCompletedChests(); err != nil {
		return nil, err
	}
	return map[string]string{"status": "CONFIRMED"}, nil
}
