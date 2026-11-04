package appstate

import (
	"activity/services/activity-service/internal/lottery/domain"
	"database/sql"
)

type GrantState struct {
	Status       string
	Candidates   []byte
	SelectedID   string
	SelectedName string
	Quantity     int64
}

func (s *State) LockReward(grantID string) (GrantState, error) {
	var g GrantState
	var selectedID, selectedName sql.NullString
	err := s.Tx.QueryRow(s.Ctx, `SELECT status,candidate_snapshot,CAST(final_item_id AS CHAR),final_item_name,quantity FROM reward_grant WHERE grant_id=? AND user_id=? AND edition_id=? FOR UPDATE`, grantID, s.User, s.Edition).Scan(&g.Status, &g.Candidates, &selectedID, &selectedName, &g.Quantity)
	if selectedID.Valid {
		g.SelectedID = selectedID.String
	}
	if selectedName.Valid {
		g.SelectedName = selectedName.String
	}
	return g, err
}

func (s *State) MarkRewardSelected(grantID string, prize domain.Prize) error {
	_, err := s.Tx.Exec(s.Ctx, `UPDATE reward_grant SET status='SELECTED',final_item_id=?,final_item_name=?,selected_at=UTC_TIMESTAMP(6) WHERE grant_id=? AND user_id=?`, prize.ID, prize.Name, grantID, s.User)
	return err
}

func (s *State) RewardSource(grantID string) (string, uint64, error) {
	var source string
	var grant uint64
	err := s.Tx.QueryRow(s.Ctx, `SELECT source_type,grant_id FROM reward_grant WHERE grant_id=? AND user_id=?`, grantID, s.User).Scan(&source, &grant)
	return source, grant, err
}
