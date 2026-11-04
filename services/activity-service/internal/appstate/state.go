package appstate

import (
	"activity/pkg/platform"
	assetsdomain "activity/services/activity-service/internal/assets/domain"
	"activity/services/activity-service/internal/participation/domain"
	"activity/services/activity-service/internal/uow"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// State is the transaction-scoped application state shared by the three
// bounded contexts. It owns the loaded wallet/participation/round snapshot and
// the small persistence primitives used by use-case services. Business
// orchestration lives in the lottery/assets/participation application packages.
type State struct {
	Projections []platform.Event
	Messages    []map[string]any
	Tx          uow.Tx
	Ctx         context.Context
	User        uint64
	Edition     uint64
	Round       int
	Version     int
	Flowers     int
	Unlocked    int
	Completed   int
	Mask        int
	Gold        int64
	Petals      int64
	Ledger      int64
	Value       int64
	Formal      int64
	Seq         int64
	Progress    int64
	Score       int64
	Reached     int64
	Rules       domain.Rules
	Now         time.Time
	Entry       int
	Biz         uint64
	Nickname    string
}

func (s *State) Load(active bool) error {
	var status string
	var start, end time.Time
	var rv int
	e := s.Tx.QueryRow(s.Ctx, `SELECT status,starts_at,ends_at,active_rule_version,UTC_TIMESTAMP(6) FROM activity_edition WHERE edition_id=? FOR SHARE`, s.Edition).Scan(&status, &start, &end, &rv, &s.Now)
	if e != nil {
		return e
	}
	if active && (status != "ACTIVE" || s.Now.Before(start) || !s.Now.Before(end)) {
		return platform.Fail(409, "ACTIVITY_NOT_ACTIVE")
	}
	e = s.Tx.QueryRow(s.Ctx, `SELECT gold_balance,petal_balance,ledger_seq FROM wallet WHERE user_id=? FOR UPDATE NOWAIT`, s.User).Scan(&s.Gold, &s.Petals, &s.Ledger)
	if errors.Is(e, uow.ErrNoRows) {
		return platform.Fail(409, "ACCOUNT_INITIALIZING")
	}
	if e != nil {
		return e
	}
	var exists int
	e = s.Tx.QueryRow(s.Ctx, `SELECT COUNT(*) FROM user_participation WHERE user_id=? AND edition_id=?`, s.User, s.Edition).Scan(&exists)
	if e != nil {
		return e
	}
	if exists == 0 {
		if !active {
			return platform.Fail(409, "NOT_PARTICIPATING")
		}
		_, e = s.Tx.Exec(s.Ctx, `INSERT INTO user_participation(user_id,edition_id,ranking_score_reached_at_sec) VALUES(?,?,?)`, s.User, s.Edition, s.Now.Unix())
		if e != nil {
			return e
		}
		_, e = s.Tx.Exec(s.Ctx, `INSERT INTO user_round(user_id,edition_id,round_no,rule_version) VALUES(?,?,1,?)`, s.User, s.Edition, rv)
		if e != nil {
			return e
		}
		e = s.Reliable("ParticipationInitialized.v1", s.User, s.Edition, map[string]string{"user_id": fmt.Sprint(s.User), "edition_id": fmt.Sprint(s.Edition), "participation_created_at_sec": fmt.Sprint(s.Now.Unix())})
		if e != nil {
			return e
		}
	}
	e = s.Tx.QueryRow(s.Ctx, `SELECT current_round_no,formal_draw_count,draw_seq,draw_progress_version,ranking_score,ranking_score_reached_at_sec FROM user_participation WHERE user_id=? AND edition_id=? FOR UPDATE`, s.User, s.Edition).Scan(&s.Round, &s.Formal, &s.Seq, &s.Progress, &s.Score, &s.Reached)
	if e != nil {
		return e
	}
	e = s.Tx.QueryRow(s.Ctx, `SELECT rule_version,flower_count,consumed_gold_equivalent,unlocked_chests,completed_chests,claimed_stage_mask FROM user_round WHERE user_id=? AND edition_id=? AND round_no=? FOR UPDATE`, s.User, s.Edition, s.Round).Scan(&s.Version, &s.Flowers, &s.Value, &s.Unlocked, &s.Completed, &s.Mask)
	if e != nil {
		return e
	}
	var rules []byte
	var hash []byte
	e = s.Tx.QueryRow(s.Ctx, `SELECT rule_json,checksum FROM activity_rule_version WHERE edition_id=? AND version=?`, s.Edition, s.Version).Scan(&rules, &hash)
	if e != nil {
		return e
	}
	var canonical any
	if e = json.Unmarshal(rules, &canonical); e != nil {
		return e
	}
	actual := sha256.Sum256(platform.JSON(canonical))
	if string(actual[:]) != string(hash) {
		return errors.New("rule checksum mismatch")
	}
	return json.Unmarshal(rules, &s.Rules)
}

func (s *State) Summary() any {
	return map[string]any{"edition_id": fmt.Sprint(s.Edition), "gold": s.Gold, "petals": s.Petals, "round_no": s.Round, "flowers": s.Flowers, "unlocked_chests": s.Unlocked, "completed_chests": s.Completed, "claimed_stage_mask": s.Mask, "score": s.Score, "rules": s.Rules}
}

func (s *State) Asset(asset string, delta int64, reason, kind string) error {
	if delta == 0 {
		return nil
	}
	wallet := assetsdomain.Wallet{Gold: s.Gold, Petals: s.Petals, LedgerSeq: s.Ledger}
	balance, err := wallet.Apply(asset, delta)
	if err != nil {
		return platform.Fail(409, "INSUFFICIENT_OR_OVERFLOWING_ASSET")
	}
	col := "petal_balance"
	if asset == assetsdomain.Gold {
		col = "gold_balance"
	}
	result, e := s.Tx.Exec(s.Ctx, fmt.Sprintf(`UPDATE wallet SET %s=%s+?,ledger_seq=ledger_seq+1,version=version+1 WHERE user_id=? AND %s>=?`, col, col, col), delta, s.User, max(0, -delta))
	if e != nil {
		return e
	}
	n, e := result.RowsAffected()
	if e != nil || n != 1 {
		return platform.Fail(409, "ASSET_CONFLICT")
	}
	s.Ledger++
	s.Entry++
	_, e = s.Tx.Exec(s.Ctx, fmt.Sprintf(`INSERT INTO %s(ledger_id,user_id,edition_id,asset_type,delta,balance_after,reason,biz_type,biz_id,entry_no,user_ledger_seq) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, platform.Shard("wallet_ledger", s.User)), platform.ID(), s.User, s.Edition, asset, delta, balance, reason, kind, s.Biz, s.Entry, s.Ledger)
	if asset == "GOLD" {
		s.Gold = balance
	} else {
		s.Petals = balance
	}
	return e
}

func (s *State) GrantValues(p domain.Prize, source string, sourceID uint64, no int, draw any) (uint64, []any) {
	id := platform.ID()
	status := "OBTAINED"
	var finalID, finalName, candidates any = p.ID, p.Name, nil
	if len(p.Choices) > 0 {
		status = "PENDING_CHOICE"
		finalID = nil
		finalName = nil
		candidates = platform.JSON(p.Choices)
	}
	var drawNo any
	if draw != nil {
		drawNo = no
	}
	return id, []any{id, s.User, s.Edition, s.Round, source, sourceID, no, draw, drawNo, p.Code, p.Name, p.Quantity, candidates, status, finalID, finalName}
}

func (s *State) Grant(p domain.Prize, source string, sourceID uint64, no int, draw any) (uint64, error) {
	id, values := s.GrantValues(p, source, sourceID, no, draw)
	_, e := s.Tx.Exec(s.Ctx, `INSERT INTO reward_grant(grant_id,user_id,edition_id,round_no,source_type,source_id,source_entry_no,draw_id,draw_no,original_reward_code,original_item_name,quantity,candidate_snapshot,status,final_item_id,final_item_name) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, values...)
	return id, e
}

func (s *State) Reliable(kind string, user, aggregate uint64, payload any) error {
	if _, e := s.Tx.Exec(s.Ctx, `INSERT INTO outbox_event(event_id,event_type,aggregate_id,user_id,payload) VALUES(?,?,?,?,?)`, platform.ID(), kind, aggregate, user, platform.JSON(payload)); e != nil {
		return e
	}
	s.Projections = append(s.Projections, platform.Event{Type: kind, Payload: platform.JSON(payload)})
	return nil
}

func (s *State) Recover(c Command) (any, bool, error) {
	var hash []byte
	var until time.Time
	var draw uint64
	e := s.Tx.QueryRow(s.Ctx, `SELECT request_hash,recover_until,result_draw_id FROM idempotency_request WHERE user_id=? AND operation_type='DRAW' AND operation_id=?`, s.User, c.Operation).Scan(&hash, &until, &draw)
	if errors.Is(e, uow.ErrNoRows) {
		return nil, false, nil
	}
	if e != nil {
		return nil, false, e
	}
	expected := RequestHash(c)
	if string(hash) != string(expected[:]) {
		return nil, false, platform.Fail(409, "IDEMPOTENCY_CONFLICT")
	}
	if !until.After(time.Now()) {
		return nil, false, platform.Fail(409, "IDEMPOTENCY_EXPIRED")
	}
	var raw []byte
	var cost, refund int64
	var requested, actual, flowers int
	e = s.Tx.QueryRow(s.Ctx, fmt.Sprintf(`SELECT reward_snapshot_json,consumed_petals,refunded_petals,requested_count,actual_count FROM %s WHERE user_id=? AND draw_id=?`, platform.Shard("draw_order", s.User)), s.User, draw).Scan(&raw, &cost, &refund, &requested, &actual)
	var steps []domain.Step
	if e == nil {
		e = json.Unmarshal(raw, &steps)
		if len(steps) > 0 {
			flowers = steps[len(steps)-1].After
		}
	}
	var results any = json.RawMessage(raw)
	if e == nil {
		rows, grantErr := s.Tx.Query(s.Ctx, `SELECT CAST(grant_id AS CHAR) FROM reward_grant WHERE user_id=? AND draw_id=? ORDER BY draw_no`, s.User, draw)
		if grantErr == nil {
			ids := []string{}
			for rows.Next() {
				var id string
				if grantErr = rows.Scan(&id); grantErr != nil {
					break
				}
				ids = append(ids, id)
			}
			rows.Close()
			if grantErr == nil && len(ids) == len(steps) {
				enriched := make([]DrawResult, 0, len(steps))
				for i, step := range steps {
					enriched = append(enriched, DrawResult{Step: step, GrantID: ids[i]})
				}
				results = enriched
			}
		}
	}
	return map[string]any{"draw_id": fmt.Sprint(draw), "results": results, "consumed_petals": cost, "refunded_petals": refund, "requested_count": requested, "actual_count": actual, "flowers": flowers}, true, e
}

func (s *State) Notify(p domain.Prize, grant uint64, source string) {
	if s.Nickname == "" || len(p.Choices) > 0 || p.ID == 1001 || p.ID == 1002 {
		return
	}
	s.Messages = append(s.Messages, map[string]any{"schema_version": 1, "event_id": fmt.Sprintf("ring:%d:%s:v1", grant, source), "event_type": "ring_awarded", "occurred_at": s.Now.UTC(), "expires_at": s.Now.Add(30 * time.Second).UTC(), "edition_id": fmt.Sprint(s.Edition), "source": source, "display_name": s.Nickname, "item_id": p.ID, "item_name": p.Name, "quantity": p.Quantity})
}

// DrawResult keeps the stored draw step JSON unchanged while adding the grant
// ID used by the H5 to resolve pending choices.
type DrawResult struct {
	domain.Step
	GrantID string `json:"grant_id"`
}

func RequestHash(c Command) [32]byte {
	return sha256.Sum256(platform.JSON([]any{c.Edition, c.Pool, c.Count, "NORMAL"}))
}
