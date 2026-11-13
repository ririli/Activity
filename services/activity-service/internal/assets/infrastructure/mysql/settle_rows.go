package mysql

import app "activity/services/activity-service/internal/assets/application"

func (s *Repository) ListPendingGrants() ([]app.PendingGrant, error) {
	rows, err := s.Tx.Query(s.Ctx, `SELECT CAST(grant_id AS CHAR),candidate_snapshot FROM reward_grant WHERE user_id=? AND edition_id=? AND status='PENDING_CHOICE' ORDER BY grant_id FOR UPDATE`, s.User, s.Edition)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []app.PendingGrant{}
	for rows.Next() {
		var p app.PendingGrant
		if err := rows.Scan(&p.ID, &p.Candidates); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Repository) OwnedItemCount(itemID uint64) (int, error) {
	var n int
	err := s.Tx.QueryRow(s.Ctx, `SELECT COUNT(*) FROM reward_grant WHERE user_id=? AND final_item_id=? AND status IN ('OBTAINED','SELECTED')`, s.User, itemID).Scan(&n)
	return n, err
}
