package mysql

func (s *Repository) ListRounds() ([]int, error) {
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

func (s *Repository) LockRoundForSettlement(round int) (flowers, mask int, rules []byte, err error) {
	err = s.Tx.QueryRow(s.Ctx, `SELECT r.flower_count,r.claimed_stage_mask,v.rule_json FROM user_round r JOIN activity_rule_version v ON v.edition_id=r.edition_id AND v.version=r.rule_version WHERE r.user_id=? AND r.edition_id=? AND r.round_no=? FOR UPDATE`, s.User, s.Edition, round).Scan(&flowers, &mask, &rules)
	return
}

func (s *Repository) ConfirmAllChests() error {
	_, err := s.Tx.Exec(s.Ctx, `UPDATE chest_open SET status='CONFIRMED',confirmed_at=UTC_TIMESTAMP(6) WHERE user_id=? AND edition_id=? AND status='OPENED'`, s.User, s.Edition)
	return err
}

func (s *Repository) RecomputeCompletedChests() error {
	_, err := s.Tx.Exec(s.Ctx, `UPDATE user_round r SET completed_chests=(SELECT COUNT(*) FROM chest_open c WHERE c.user_id=r.user_id AND c.edition_id=r.edition_id AND c.round_no=r.round_no AND c.status='CONFIRMED') WHERE r.user_id=? AND r.edition_id=?`, s.User, s.Edition)
	return err
}
