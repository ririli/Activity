package appstate

import "activity/pkg/platform"

func (s *State) InitializeAccount(eventID uint64) error {
	res, err := s.Tx.Exec(s.Ctx, `INSERT IGNORE INTO inbox_event(consumer_name,event_id) VALUES('account-init',?)`, eventID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil
	}
	res, err = s.Tx.Exec(s.Ctx, `INSERT IGNORE INTO wallet(user_id,gold_balance,petal_balance,ledger_seq) VALUES(?,10000000,100000000,2)`, s.User)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil
	}
	for i, asset := range []string{"GOLD", "PETAL"} {
		v := int64(10000000)
		if i == 1 {
			v = 100000000
		}
		if _, err = s.Tx.Exec(s.Ctx, "INSERT INTO "+platform.Shard("wallet_ledger", s.User)+
			`(ledger_id,user_id,asset_type,delta,balance_after,reason,biz_type,biz_id,entry_no,user_ledger_seq) VALUES(?,?,?,?,?,'INITIALIZATION','ACCOUNT',?,?,?)`,
			platform.ID(), s.User, asset, v, v, s.User, i+1, i+1); err != nil {
			return err
		}
	}
	return nil
}
