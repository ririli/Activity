package application

import (
	"activity/services/activity-service/internal/fault"
	lottery "activity/services/activity-service/internal/lottery/domain"
	partdomain "activity/services/activity-service/internal/participation/domain"
	"encoding/json"
	"errors"
	"fmt"
)

func openChest(s *work) (any, error) {
	id, grant, raw, err := s.session.Participation.FindOpenChest()
	if err == nil {
		return map[string]any{"chest_id": fmt.Sprint(id), "grant_id": fmt.Sprint(grant), "reward": json.RawMessage(raw)}, nil
	}
	if !errors.Is(err, fault.ErrNotFound) {
		return nil, err
	}
	if s.progress.Completed >= s.progress.Unlocked {
		return nil, fault.New("NO_CHEST_AVAILABLE")
	}
	used, err := s.session.Participation.UsedChestCodes()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, code := range used {
		seen[code] = true
	}
	choices := []lottery.Prize{}
	for _, p := range s.rules.Chests {
		if !seen[p.Code] {
			choices = append(choices, p)
		}
	}
	if len(choices) == 0 {
		return nil, fault.New("NO_CHEST_AVAILABLE")
	}
	index, err := s.runtime.Random(int64(len(choices)))
	if err != nil {
		return nil, err
	}
	p := choices[index]
	id, err = s.newID()
	if err != nil {
		return nil, err
	}
	grant, err = s.Grant(p, "CHEST", id, 1)
	if err != nil {
		return nil, err
	}
	kind := "ITEM"
	if len(p.Choices) > 0 {
		kind = "CHOICE"
	}
	if err = s.session.Participation.InsertChest(id, s.progress.Completed+1, p.Code, kind, encode(p), grant); err != nil {
		return nil, err
	}
	return map[string]any{"chest_id": fmt.Sprint(id), "grant_id": fmt.Sprint(grant), "reward": p}, nil
}

func confirmChest(s *work, id string) (any, error) {
	status, reward, round, err := s.session.Participation.LockChest(id)
	if err != nil {
		return nil, err
	}
	if status == "CONFIRMED" {
		return map[string]string{"status": status}, nil
	}
	if err = (partdomain.Chest{Status: status, RewardStatus: reward, Round: round}).CheckConfirm(s.progress.Round); err != nil {
		return nil, err
	}
	if err = s.session.Participation.ConfirmChest(id); err != nil {
		return nil, err
	}
	if err = s.session.Participation.IncrementCompletedChests(); err != nil {
		return nil, err
	}
	return map[string]string{"status": "CONFIRMED"}, nil
}
