package mysql

import (
	"activity/services/activity-service/internal/infrastructure/sqlkit"
	domain "activity/services/activity-service/internal/lottery/domain"
	"context"
)

type Repository struct {
	Tx                 sqlkit.Tx
	Ctx                context.Context
	User, Edition, Biz uint64
	Round              int
	Progress           int64
	NewID              func(context.Context) (uint64, error)
}

func (s *Repository) FindPreview() (id uint64, raw []byte, status string, cost int64, err error) {
	err = s.Tx.QueryRow(s.Ctx, `SELECT preview_id,result_json,status,required_petals FROM preview_order WHERE user_id=? AND edition_id=?`, s.User, s.Edition).Scan(&id, &raw, &status, &cost)
	return
}

func (s *Repository) LockPreview() (domain.Preview, error) {
	var p domain.Preview
	err := s.Tx.QueryRow(s.Ctx, `SELECT preview_id,round_no,start_progress_version,status,result_json,expected_progress_json,result_hash,COALESCE(confirmed_draw_id,0) FROM preview_order WHERE user_id=? AND edition_id=? FOR UPDATE`, s.User, s.Edition).Scan(&p.ID, &p.Round, &p.Version, &p.Status, &p.Raw, &p.Expected, &p.Hash, &p.DrawID)
	return p, err
}

func (s *Repository) InsertPreview(id uint64, round, version, actual int, cost int64, snapshot, raw, expected, hash []byte) error {
	_, err := s.Tx.Exec(s.Ctx, `INSERT INTO preview_order(preview_id,user_id,edition_id,round_no,rule_version,start_progress_snapshot,start_progress_version,actual_count,required_petals,result_json,expected_progress_json,result_hash) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, id, s.User, s.Edition, round, version, snapshot, s.Progress, actual, cost, raw, expected, hash)
	return err
}

func (s *Repository) ConfirmPreview(id, drawID uint64) error {
	_, err := s.Tx.Exec(s.Ctx, `UPDATE preview_order SET status='CONFIRMED',confirmed_draw_id=?,confirmed_at=UTC_TIMESTAMP(6) WHERE preview_id=?`, drawID, id)
	return err
}

func (s *Repository) AbandonPreview() (int64, error) {
	res, err := s.Tx.Exec(s.Ctx, `UPDATE preview_order SET status='ABANDONED',abandoned_at=UTC_TIMESTAMP(6) WHERE user_id=? AND edition_id=? AND status='PENDING'`, s.User, s.Edition)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Repository) PreviewStatus() (string, error) {
	var status string
	err := s.Tx.QueryRow(s.Ctx, `SELECT status FROM preview_order WHERE user_id=? AND edition_id=?`, s.User, s.Edition).Scan(&status)
	return status, err
}
