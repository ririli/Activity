package mysql

import (
	"activity/pkg/platform"
	app "activity/services/activity-service/internal/assets/application"
	"activity/services/activity-service/internal/infrastructure/sqlkit"
)

func (s *Repository) SaveGrants(grants []app.Grant) error {
	rows := make([][]any, 0, len(grants))
	for _, g := range grants {
		p := g.Prize
		status := "OBTAINED"
		var finalID, finalName, candidates any = p.ID, p.Name, nil
		if len(p.Choices) > 0 {
			status = "PENDING_CHOICE"
			finalID = nil
			finalName = nil
			candidates = platform.JSON(p.Choices)
		}
		var draw, no, round any
		if g.Round != 0 {
			round = g.Round
		}
		if g.DrawID != 0 {
			draw = g.DrawID
			no = g.No
		}
		rows = append(rows, []any{g.ID, s.User, s.Edition, round, g.Source, g.SourceID, g.No, draw, no, p.Code, p.Name, p.Quantity, candidates, status, finalID, finalName})
	}
	return sqlkit.BatchInsert(s.Ctx, s.Tx, "reward_grant", []string{"grant_id", "user_id", "edition_id", "round_no", "source_type", "source_id", "source_entry_no", "draw_id", "draw_no", "original_reward_code", "original_item_name", "quantity", "candidate_snapshot", "status", "final_item_id", "final_item_name"}, rows)
}
