package appstate

type PendingGrant struct {
	ID         string
	Candidates []byte
}

func (s *State) ListRounds() ([]int, error) {
	rows, err := s.Tx.Query(s.Ctx, `SELECT round_no FROM user_round WHERE user_id=? AND edition_id=? ORDER BY round_no`, s.User, s.Edition)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	rounds := []int{}
	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		rounds = append(rounds, n)
	}
	return rounds, rows.Err()
}

func (s *State) LockRoundForSettlement(round int) (flowers, mask int, rules []byte, err error) {
	err = s.Tx.QueryRow(s.Ctx, `SELECT r.flower_count,r.claimed_stage_mask,v.rule_json FROM user_round r JOIN activity_rule_version v ON v.edition_id=r.edition_id AND v.version=r.rule_version WHERE r.user_id=? AND r.edition_id=? AND r.round_no=? FOR UPDATE`, s.User, s.Edition, round).Scan(&flowers, &mask, &rules)
	return
}

func (s *State) ListPendingGrants() ([]PendingGrant, error) {
	rows, err := s.Tx.Query(s.Ctx, `SELECT CAST(grant_id AS CHAR),candidate_snapshot FROM reward_grant WHERE user_id=? AND edition_id=? AND status='PENDING_CHOICE' ORDER BY grant_id FOR UPDATE`, s.User, s.Edition)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PendingGrant{}
	for rows.Next() {
		var p PendingGrant
		if err := rows.Scan(&p.ID, &p.Candidates); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *State) OwnedItemCount(itemID uint64) (int, error) {
	var n int
	err := s.Tx.QueryRow(s.Ctx, `SELECT COUNT(*) FROM reward_grant WHERE user_id=? AND final_item_id=? AND status IN ('OBTAINED','SELECTED')`, s.User, itemID).Scan(&n)
	return n, err
}

func (s *State) ConfirmAllChests() error {
	_, err := s.Tx.Exec(s.Ctx, `UPDATE chest_open SET status='CONFIRMED',confirmed_at=UTC_TIMESTAMP(6) WHERE user_id=? AND edition_id=? AND status='OPENED'`, s.User, s.Edition)
	return err
}

func (s *State) RecomputeCompletedChests() error {
	_, err := s.Tx.Exec(s.Ctx, `UPDATE user_round r SET completed_chests=(SELECT COUNT(*) FROM chest_open c WHERE c.user_id=r.user_id AND c.edition_id=r.edition_id AND c.round_no=r.round_no AND c.status='CONFIRMED') WHERE r.user_id=? AND r.edition_id=?`, s.User, s.Edition)
	return err
}
