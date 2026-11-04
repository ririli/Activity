package appstate

import (
	"activity/pkg/platform"
	"time"
)

func (s *State) CloseEdition() error {
	var end, now time.Time
	if err := s.Tx.QueryRow(s.Ctx, `SELECT ends_at,UTC_TIMESTAMP(6) FROM activity_edition WHERE edition_id=? FOR UPDATE`, s.Edition).Scan(&end, &now); err != nil {
		return err
	}
	if now.Before(end) {
		return platform.Fail(409, "ACTIVITY_NOT_ENDED")
	}
	_, err := s.Tx.Exec(s.Ctx, `UPDATE activity_edition SET status='ENDED' WHERE edition_id=?`, s.Edition)
	return err
}
