package appstate

func (s *State) FindPurchase(request string) (id uint64, kind, product string, petals int64, err error) {
	err = s.Tx.QueryRow(s.Ctx, `SELECT order_id,purchase_type,product_code,petals_granted FROM purchase_order WHERE user_id=? AND request_id=?`, s.User, request).Scan(&id, &kind, &product, &petals)
	return
}

func (s *State) CountPurchaseQuota(product string) (int, error) {
	var n int
	err := s.Tx.QueryRow(s.Ctx, `SELECT COUNT(*) FROM purchase_quota WHERE user_id=? AND edition_id=? AND product_code=?`, s.User, s.Edition, product).Scan(&n)
	return n, err
}

func (s *State) InsertPurchase(request, kind, product string, cost, petals int64, version int) error {
	_, err := s.Tx.Exec(s.Ctx, `INSERT INTO purchase_order(order_id,user_id,edition_id,request_id,purchase_type,product_code,gold_cost,petals_granted,rule_version) VALUES(?,?,?,?,?,?,?,?,?)`, s.Biz, s.User, s.Edition, request, kind, product, cost, petals, version)
	return err
}

func (s *State) InsertPurchaseQuota(product string) error {
	_, err := s.Tx.Exec(s.Ctx, `INSERT INTO purchase_quota(user_id,edition_id,product_code,order_id) VALUES(?,?,?,?)`, s.User, s.Edition, product, s.Biz)
	return err
}
