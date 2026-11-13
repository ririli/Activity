package application

import (
	lottery "activity/services/activity-service/internal/lottery/domain"
)

func selectReward(s *work, id, item string) (any, error) {
	p, source, grant, err := s.assets.Select(id, item)
	if err != nil {
		return nil, err
	}
	prize := lottery.Prize{ID: p.ID, Code: p.Code, Name: p.Name, Quantity: p.Quantity}
	if source == "DAY_DRAW" || source == "NIGHT_DRAW" {
		s.Notify(prize, grant, "normal_draw_selection")
	}
	if source == "PREVIEW" {
		s.Notify(prize, grant, "pay_later_selection")
	}
	return map[string]any{"grant_id": id, "item_id": item, "item_name": p.Name, "quantity": p.Quantity}, nil
}
