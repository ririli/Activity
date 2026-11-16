package mysql

import (
	"activity/pkg/platform"
	app "activity/services/activity-service/internal/assets/application"
	domain "activity/services/activity-service/internal/assets/domain"
	"activity/services/activity-service/internal/infrastructure/sqlkit"
	"context"
	"database/sql"
	"encoding/json"
)

type Repository struct {
	Tx                 sqlkit.Tx
	Ctx                context.Context
	User, Edition, Biz uint64
	Round              int
	Progress           int64
	NewID              func(context.Context) (uint64, error)
}

func (s *Repository) FindPurchase(request string) (id, edition uint64, kind, product string, petals int64, err error) {
	err = s.Tx.QueryRow(s.Ctx, `SELECT order_id,edition_id,purchase_type,product_code,petals_granted FROM purchase_order WHERE user_id=? AND request_id=?`, s.User, request).Scan(&id, &edition, &kind, &product, &petals)
	return
}

func (s *Repository) CountPurchaseQuota(product string) (int, error) {
	var n int
	err := s.Tx.QueryRow(s.Ctx, `SELECT COUNT(*) FROM purchase_quota WHERE user_id=? AND edition_id=? AND product_code=?`, s.User, s.Edition, product).Scan(&n)
	return n, err
}

func (s *Repository) InsertPurchase(request, kind, product string, cost, petals int64, version int) error {
	_, err := s.Tx.Exec(s.Ctx, `INSERT INTO purchase_order(order_id,user_id,edition_id,request_id,purchase_type,product_code,gold_cost,petals_granted,rule_version) VALUES(?,?,?,?,?,?,?,?,?)`, s.Biz, s.User, s.Edition, request, kind, product, cost, petals, version)
	return err
}

func (s *Repository) InsertPurchaseQuota(product string) error {
	_, err := s.Tx.Exec(s.Ctx, `INSERT INTO purchase_quota(user_id,edition_id,product_code,order_id) VALUES(?,?,?,?)`, s.User, s.Edition, product, s.Biz)
	return err
}

func (s *Repository) LockReward(grantID string) (app.GrantState, error) {
	var g app.GrantState
	var selectedID, selectedName sql.NullString
	err := s.Tx.QueryRow(s.Ctx, `SELECT status,candidate_snapshot,CAST(final_item_id AS CHAR),final_item_name,quantity FROM reward_grant WHERE grant_id=? AND user_id=? AND edition_id=? FOR UPDATE`, grantID, s.User, s.Edition).Scan(&g.Reward.Status, &g.Candidates, &selectedID, &selectedName, &g.Reward.Quantity)
	if selectedID.Valid {
		g.Reward.SelectedID = selectedID.String
	}
	if selectedName.Valid {
		g.Reward.SelectedName = selectedName.String
	}
	if err == nil && len(g.Candidates) > 0 {
		err = json.Unmarshal(g.Candidates, &g.Reward.Candidates)
	}
	return g, err
}

func (s *Repository) MarkRewardSelected(grantID string, prize domain.Prize) error {
	_, err := s.Tx.Exec(s.Ctx, `UPDATE reward_grant SET status='SELECTED',final_item_id=?,final_item_name=?,selected_at=UTC_TIMESTAMP(6) WHERE grant_id=? AND user_id=?`, prize.ID, prize.Name, grantID, s.User)
	return err
}

func (s *Repository) RewardSource(grantID string) (string, uint64, error) {
	var source string
	var grant uint64
	err := s.Tx.QueryRow(s.Ctx, `SELECT source_type,grant_id FROM reward_grant WHERE grant_id=? AND user_id=?`, grantID, s.User).Scan(&source, &grant)
	return source, grant, err
}

func (s *Repository) InitializeAccount(eventID uint64, w domain.Wallet) error {
	res, err := s.Tx.Exec(s.Ctx, `INSERT IGNORE INTO inbox_event(consumer_name,event_id) VALUES('account-init',?)`, eventID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil
	}
	res, err = s.Tx.Exec(s.Ctx, `INSERT IGNORE INTO wallet(user_id,gold_balance,petal_balance,ledger_seq) VALUES(?,?,?,?)`, s.User, w.Gold, w.Petals, w.LedgerSeq)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil
	}
	for i, asset := range []string{"GOLD", "PETAL"} {
		v := w.Gold
		if i == 1 {
			v = w.Petals
		}
		id, err := s.NewID(s.Ctx)
		if err != nil {
			return err
		}
		if _, err = s.Tx.Exec(s.Ctx, "INSERT INTO "+platform.Shard("wallet_ledger", s.User)+
			`(ledger_id,user_id,asset_type,delta,balance_after,reason,biz_type,biz_id,entry_no,user_ledger_seq) VALUES(?,?,?,?,?,'INITIALIZATION','ACCOUNT',?,?,?)`,
			id, s.User, asset, v, v, s.User, i+1, i+1); err != nil {
			return err
		}
	}
	return nil
}

func (s *Repository) FindRankingGrant(award uint64) (uint64, error) {
	var grant uint64
	err := s.Tx.QueryRow(s.Ctx, `SELECT grant_id FROM reward_grant WHERE user_id=? AND source_type='RANKING' AND source_id=?`, s.User, award).Scan(&grant)
	return grant, err
}

func (s *Repository) LockWallet() error {
	var user uint64
	return s.Tx.QueryRow(s.Ctx, `SELECT user_id FROM wallet WHERE user_id=? FOR UPDATE`, s.User).Scan(&user)
}

func (s *Repository) InsertRankingGrant(grant, award uint64, item, name string, itemID uint64, quantity int64) error {
	_, err := s.Tx.Exec(s.Ctx, `INSERT INTO reward_grant(grant_id,user_id,edition_id,source_type,source_id,source_entry_no,original_reward_code,original_item_name,quantity,status,final_item_id,final_item_name) VALUES(?,?,?,'RANKING',?,1,?,?,?,'OBTAINED',?,?)`, grant, s.User, s.Edition, award, item, name, quantity, itemID, name)
	return err
}

func (s *Repository) InsertPreviewTopUpFor(previewID, orderID uint64, request string, cost, petals int64, version int) error {
	_, err := s.Tx.Exec(s.Ctx, `INSERT INTO purchase_order(order_id,user_id,edition_id,request_id,purchase_type,product_code,gold_cost,petals_granted,rule_version,parent_confirm_id) VALUES(?,?,?,?,'PREVIEW_TOP_UP','top-up',?,?,?,?)`, orderID, s.User, s.Edition, request, cost, petals, version, previewID)
	return err
}
