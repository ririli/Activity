package mysql

import (
	"activity/pkg/platform"
	app "activity/services/activity-service/internal/lottery/application"
	domain "activity/services/activity-service/internal/lottery/domain"
	"encoding/json"
	"fmt"
)

func (s *Repository) FindDraw(operation string) (app.Recovery, error) {
	var v app.Recovery
	var id uint64
	err := s.Tx.QueryRow(s.Ctx, `SELECT request_hash,recover_until,result_draw_id FROM idempotency_request WHERE user_id=? AND operation_type='DRAW' AND operation_id=?`, s.User, operation).Scan(&v.Hash, &v.Until, &id)
	if err != nil {
		return v, err
	}
	v.Result, err = s.LoadDraw(id)
	return v, err
}

func (s *Repository) LoadDraw(id uint64) (app.DrawResult, error) {
	var result app.DrawResult
	var raw []byte
	err := s.Tx.QueryRow(s.Ctx, fmt.Sprintf(`SELECT reward_snapshot_json,consumed_petals,refunded_petals,requested_count,actual_count FROM %s WHERE user_id=? AND draw_id=?`, platform.Shard("draw_order", s.User)), s.User, id).Scan(&raw, &result.Consumed, &result.Refunded, &result.Requested, &result.Actual)
	if err != nil {
		return result, err
	}
	var steps []domain.Step
	if err = json.Unmarshal(raw, &steps); err != nil {
		return result, err
	}
	rows, err := s.Tx.Query(s.Ctx, `SELECT CAST(grant_id AS CHAR) FROM reward_grant WHERE user_id=? AND draw_id=? ORDER BY draw_no`, s.User, id)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return result, err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return result, err
	}
	if len(ids) != len(steps) {
		return result, fmt.Errorf("draw/reward count mismatch")
	}
	result.DrawID = fmt.Sprint(id)
	result.Results = make([]app.DrawStep, 0, len(steps))
	for i, step := range steps {
		result.Results = append(result.Results, app.DrawStep{Step: step, GrantID: ids[i]})
		result.Flowers = step.After
	}
	return result, nil
}
