package mysql

import domain "activity/services/activity-service/internal/participation/domain"

func (s *Repository) Gate() (domain.Edition, error) {
	var e domain.Edition
	err := s.Tx.QueryRow(s.Ctx, `SELECT status,starts_at,ends_at,active_rule_version,UTC_TIMESTAMP(6),hourly_ranking_enabled FROM activity_edition WHERE edition_id=? FOR SHARE`, s.Edition).Scan(&e.Status, &e.Start, &e.End, &e.RuleVersion, &e.Now, &e.HourlyRanking)
	return e, err
}
func (s *Repository) Exists() (bool, error) {
	var n int
	err := s.Tx.QueryRow(s.Ctx, `SELECT COUNT(*) FROM user_participation WHERE user_id=? AND edition_id=?`, s.User, s.Edition).Scan(&n)
	return n > 0, err
}
func (s *Repository) Initialize(version int, now int64) error {
	if _, err := s.Tx.Exec(s.Ctx, `INSERT INTO user_participation(user_id,edition_id,ranking_score_reached_at_sec) VALUES(?,?,?)`, s.User, s.Edition, now); err != nil {
		return err
	}
	_, err := s.Tx.Exec(s.Ctx, `INSERT INTO user_round(user_id,edition_id,round_no,rule_version) VALUES(?,?,1,?)`, s.User, s.Edition, version)
	return err
}
func (s *Repository) LoadProgress() (domain.Progress, error) {
	var p domain.Progress
	err := s.Tx.QueryRow(s.Ctx, `SELECT current_round_no,formal_draw_count,draw_seq,draw_progress_version,ranking_score,ranking_score_reached_at_sec FROM user_participation WHERE user_id=? AND edition_id=? FOR UPDATE`, s.User, s.Edition).Scan(&p.Round, &p.Formal, &p.Seq, &p.Progress, &p.Score, &p.Reached)
	if err != nil {
		return p, err
	}
	err = s.Tx.QueryRow(s.Ctx, `SELECT rule_version,flower_count,consumed_gold_equivalent,unlocked_chests,completed_chests,claimed_stage_mask FROM user_round WHERE user_id=? AND edition_id=? AND round_no=? FOR UPDATE`, s.User, s.Edition, p.Round).Scan(&p.Version, &p.Flowers, &p.Value, &p.Unlocked, &p.Completed, &p.Mask)
	return p, err
}
func (s *Repository) Rule(version int) ([]byte, []byte, error) {
	var raw, hash []byte
	err := s.Tx.QueryRow(s.Ctx, `SELECT rule_json,checksum FROM activity_rule_version WHERE edition_id=? AND version=?`, s.Edition, version).Scan(&raw, &hash)
	return raw, hash, err
}
func (s *Repository) SaveDrawProgress(p domain.Progress) error {
	status := "ACTIVE"
	if p.Flowers == 18 {
		status = "FULL"
	}
	if _, err := s.Tx.Exec(s.Ctx, `UPDATE user_round SET flower_count=?,consumed_gold_equivalent=?,unlocked_chests=?,status=? WHERE user_id=? AND edition_id=? AND round_no=?`, p.Flowers, p.Value, p.Unlocked, status, s.User, s.Edition, p.Round); err != nil {
		return err
	}
	_, err := s.Tx.Exec(s.Ctx, `UPDATE user_participation SET formal_draw_count=formal_draw_count+1,draw_seq=?,draw_progress_version=draw_progress_version+1,version=version+1,ranking_score=?,ranking_score_reached_at_sec=? WHERE user_id=? AND edition_id=?`, p.Seq, p.Score, p.Reached, s.User, s.Edition)
	return err
}
