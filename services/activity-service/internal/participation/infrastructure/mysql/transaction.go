package mysql

import (
	"activity/services/activity-service/internal/infrastructure/sqlkit"
	domain "activity/services/activity-service/internal/participation/domain"
	"context"
)

type Repository struct {
	Tx                 sqlkit.Tx
	Ctx                context.Context
	User, Edition, Biz uint64
	Round              int
	Progress           int64
}

func (s *Repository) FindStageClaim(stage int) (uint64, error) {
	var grant uint64
	err := s.Tx.QueryRow(s.Ctx, `SELECT reward_grant_id FROM stage_claim WHERE user_id=? AND edition_id=? AND round_no=? AND stage_code=?`, s.User, s.Edition, s.Round, stage).Scan(&grant)
	return grant, err
}

func (s *Repository) InsertStageClaim(claimID uint64, stage int, grant uint64, source string) error {
	_, err := s.Tx.Exec(s.Ctx, `INSERT INTO stage_claim(claim_id,user_id,edition_id,round_no,stage_code,reward_grant_id,claim_source) VALUES(?,?,?,?,?,?,?)`, claimID, s.User, s.Edition, s.Round, stage, grant, source)
	return err
}

func (s *Repository) UpdateRoundMask(mask int) error {
	_, err := s.Tx.Exec(s.Ctx, `UPDATE user_round SET claimed_stage_mask=? WHERE user_id=? AND edition_id=? AND round_no=?`, mask, s.User, s.Edition, s.Round)
	return err
}

func (s *Repository) FindOpenChest() (id, grant uint64, raw []byte, err error) {
	err = s.Tx.QueryRow(s.Ctx, `SELECT chest_id,reward_grant_id,candidate_snapshot FROM chest_open WHERE user_id=? AND edition_id=? AND round_no=? AND status='OPENED'`, s.User, s.Edition, s.Round).Scan(&id, &grant, &raw)
	return
}

func (s *Repository) UsedChestCodes() ([]string, error) {
	rows, err := s.Tx.Query(s.Ctx, `SELECT candidate_code FROM chest_open WHERE user_id=? AND edition_id=? AND round_no=?`, s.User, s.Edition, s.Round)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	codes := []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

func (s *Repository) InsertChest(id uint64, chestNo int, code, kind string, snapshot []byte, grant uint64) error {
	_, err := s.Tx.Exec(s.Ctx, `INSERT INTO chest_open(chest_id,user_id,edition_id,round_no,chest_no,candidate_code,reward_type,candidate_snapshot,reward_grant_id) VALUES(?,?,?,?,?,?,?,?,?)`, id, s.User, s.Edition, s.Round, chestNo, code, kind, snapshot, grant)
	return err
}

func (s *Repository) LockChest(id string) (status, reward string, round int, err error) {
	err = s.Tx.QueryRow(s.Ctx, `SELECT c.status,r.status,c.round_no FROM chest_open c JOIN reward_grant r ON c.reward_grant_id=r.grant_id WHERE c.chest_id=? AND c.user_id=? AND c.edition_id=? FOR UPDATE`, id, s.User, s.Edition).Scan(&status, &reward, &round)
	return
}

func (s *Repository) ConfirmChest(id string) error {
	_, err := s.Tx.Exec(s.Ctx, `UPDATE chest_open SET status='CONFIRMED',confirmed_at=UTC_TIMESTAMP(6) WHERE chest_id=?`, id)
	return err
}

func (s *Repository) IncrementCompletedChests() error {
	_, err := s.Tx.Exec(s.Ctx, `UPDATE user_round SET completed_chests=completed_chests+1 WHERE user_id=? AND edition_id=? AND round_no=?`, s.User, s.Edition, s.Round)
	return err
}

func (s *Repository) PendingStageChoices(round int) (int, error) {
	var n int
	err := s.Tx.QueryRow(s.Ctx, `SELECT COUNT(*) FROM reward_grant WHERE user_id=? AND edition_id=? AND round_no=? AND source_type='STAGE' AND status='PENDING_CHOICE'`, s.User, s.Edition, round).Scan(&n)
	return n, err
}

func (s *Repository) CloseRound(round int) error {
	_, err := s.Tx.Exec(s.Ctx, `UPDATE user_round SET status='COMPLETED',completed_at=UTC_TIMESTAMP(6) WHERE user_id=? AND edition_id=? AND round_no=?`, s.User, s.Edition, round)
	return err
}

func (s *Repository) InsertRound(round, version int) error {
	_, err := s.Tx.Exec(s.Ctx, `INSERT INTO user_round(user_id,edition_id,round_no,rule_version) VALUES(?,?,?,?)`, s.User, s.Edition, round, version)
	return err
}

func (s *Repository) IncrementParticipationRound() error {
	_, err := s.Tx.Exec(s.Ctx, `UPDATE user_participation SET current_round_no=current_round_no+1,version=version+1 WHERE user_id=? AND edition_id=?`, s.User, s.Edition)
	return err
}

func (s *Repository) LockEditionForClose() (domain.Edition, error) {
	var e domain.Edition
	err := s.Tx.QueryRow(s.Ctx, `SELECT ends_at,UTC_TIMESTAMP(6) FROM activity_edition WHERE edition_id=? FOR UPDATE`, s.Edition).Scan(&e.End, &e.Now)
	return e, err
}
func (s *Repository) MarkEditionEnded() error {
	_, err := s.Tx.Exec(s.Ctx, `UPDATE activity_edition SET status='ENDED' WHERE edition_id=?`, s.Edition)
	return err
}
