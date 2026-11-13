package mysql

import (
	"activity/pkg/platform"
	app "activity/services/activity-service/internal/assets/application"
	domain "activity/services/activity-service/internal/assets/domain"
	"activity/services/activity-service/internal/fault"
	"fmt"
)

func (s *Repository) LoadWallet() (domain.Wallet, error) {
	var w domain.Wallet
	err := s.Tx.QueryRow(s.Ctx, `SELECT gold_balance,petal_balance,ledger_seq FROM wallet WHERE user_id=? FOR UPDATE NOWAIT`, s.User).Scan(&w.Gold, &w.Petals, &w.LedgerSeq)
	return w, err
}
func (s *Repository) ApplyChange(c app.Change) error {
	col := "petal_balance"
	if c.Asset == domain.Gold {
		col = "gold_balance"
	}
	res, err := s.Tx.Exec(s.Ctx, fmt.Sprintf(`UPDATE wallet SET %s=%s+?,ledger_seq=ledger_seq+1,version=version+1 WHERE user_id=? AND %s>=?`, col, col, col), c.Delta, s.User, max(0, -c.Delta))
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fault.New("ASSET_CONFLICT")
	}
	_, err = s.Tx.Exec(s.Ctx, fmt.Sprintf(`INSERT INTO %s(ledger_id,user_id,edition_id,asset_type,delta,balance_after,reason,biz_type,biz_id,entry_no,user_ledger_seq) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, platform.Shard("wallet_ledger", s.User)), platform.ID(), s.User, s.Edition, c.Asset, c.Delta, c.Balance, c.Reason, c.Kind, s.Biz, c.Entry, c.Ledger)
	return err
}
