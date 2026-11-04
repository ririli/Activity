package appstate

func (s *State) FindRankingGrant(award uint64) (uint64, error) {
	var grant uint64
	err := s.Tx.QueryRow(s.Ctx, `SELECT grant_id FROM reward_grant WHERE user_id=? AND source_type='RANKING' AND source_id=?`, s.User, award).Scan(&grant)
	return grant, err
}

func (s *State) LockWallet() error {
	var user uint64
	return s.Tx.QueryRow(s.Ctx, `SELECT user_id FROM wallet WHERE user_id=? FOR UPDATE`, s.User).Scan(&user)
}

func (s *State) InsertRankingGrant(grant, award uint64, item, name string, itemID uint64, quantity int64) error {
	_, err := s.Tx.Exec(s.Ctx, `INSERT INTO reward_grant(grant_id,user_id,edition_id,source_type,source_id,source_entry_no,original_reward_code,original_item_name,quantity,status,final_item_id,final_item_name) VALUES(?,?,?,'RANKING',?,1,?,?,?,'OBTAINED',?,?)`, grant, s.User, s.Edition, award, item, name, quantity, itemID, name)
	return err
}

func (s *State) AddRankingReceipt(award uint64, payload any) error {
	return s.Reliable("RankingAwardGranted.v1", s.User, award, payload)
}
