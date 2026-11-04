package appstate

import (
	"activity/pkg/platform"
	"crypto/sha256"
	"encoding/json"
)

func (s *State) FindPreview() (id uint64, raw []byte, status string, err error) {
	err = s.Tx.QueryRow(s.Ctx, `SELECT preview_id,result_json,status FROM preview_order WHERE user_id=? AND edition_id=?`, s.User, s.Edition).Scan(&id, &raw, &status)
	return
}

type PreviewLock struct {
	ID       uint64
	Round    int
	Version  int64
	Status   string
	Raw      []byte
	Expected []byte
	Hash     []byte
	DrawID   uint64
}

func (s *State) LockPreview() (PreviewLock, error) {
	var p PreviewLock
	err := s.Tx.QueryRow(s.Ctx, `SELECT preview_id,round_no,start_progress_version,status,result_json,expected_progress_json,result_hash,COALESCE(confirmed_draw_id,0) FROM preview_order WHERE user_id=? AND edition_id=? FOR UPDATE`, s.User, s.Edition).Scan(&p.ID, &p.Round, &p.Version, &p.Status, &p.Raw, &p.Expected, &p.Hash, &p.DrawID)
	return p, err
}

func (s *State) InsertPreview(id uint64, round, version, actual int, cost int64, snapshot, raw, expected, hash []byte) error {
	_, err := s.Tx.Exec(s.Ctx, `INSERT INTO preview_order(preview_id,user_id,edition_id,round_no,rule_version,start_progress_snapshot,start_progress_version,actual_count,required_petals,result_json,expected_progress_json,result_hash) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, id, s.User, s.Edition, round, version, snapshot, s.Progress, actual, cost, raw, expected, hash)
	return err
}

func (s *State) ConfirmPreview(id, drawID uint64) error {
	_, err := s.Tx.Exec(s.Ctx, `UPDATE preview_order SET status='CONFIRMED',confirmed_draw_id=?,confirmed_at=UTC_TIMESTAMP(6) WHERE preview_id=?`, drawID, id)
	return err
}

func (s *State) AbandonPreview() (int64, error) {
	res, err := s.Tx.Exec(s.Ctx, `UPDATE preview_order SET status='ABANDONED',abandoned_at=UTC_TIMESTAMP(6) WHERE user_id=? AND edition_id=? AND status='PENDING'`, s.User, s.Edition)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *State) PreviewStatus() (string, error) {
	var status string
	err := s.Tx.QueryRow(s.Ctx, `SELECT status FROM preview_order WHERE user_id=? AND edition_id=?`, s.User, s.Edition).Scan(&status)
	return status, err
}

func (s *State) InsertPreviewTopUpFor(previewID, orderID uint64, request string, cost, petals int64, version int) error {
	_, err := s.Tx.Exec(s.Ctx, `INSERT INTO purchase_order(order_id,user_id,edition_id,request_id,purchase_type,product_code,gold_cost,petals_granted,rule_version,parent_confirm_id) VALUES(?,?,?,?,'PREVIEW_TOP_UP','top-up',?,?,?,?)`, orderID, s.User, s.Edition, request, cost, petals, version, previewID)
	return err
}

func CanonicalHash(raw []byte) [32]byte {
	var v any
	if e := json.Unmarshal(raw, &v); e != nil {
		panic(e)
	}
	return sha256.Sum256(platform.JSON(v))
}
