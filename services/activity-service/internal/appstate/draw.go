package appstate

import (
	"activity/pkg/platform"
	"time"
)

type DrawWrite struct {
	DrawID       uint64
	Round        int
	Version      int
	Mode         string
	PreviewID    uint64
	Pool         string
	Requested    int
	Actual       int
	UnitCost     int64
	Charged      int64
	Refund       int64
	Consumed     int64
	FlowersAdded int
	FlowersAfter int
	GoldValue    int64
	Unlocked     int
	Status       string
	Snapshot     []byte
	Seq          int64
	Occurred     time.Time
	Operation    string
	RequestHash  [32]byte
	Idempotent   bool
	Details      [][]any
	Flowers      [][]any
	Grants       [][]any
}

// PersistDraw writes the committed draw facts, progress and idempotency record
// in the caller's transaction. Business decisions stay in lottery/application.
func (s *State) PersistDraw(w DrawWrite) error {
	var preview any
	if w.PreviewID != 0 {
		preview = w.PreviewID
	}
	if _, e := s.Tx.Exec(s.Ctx, "INSERT INTO "+platform.Shard("draw_order", s.User)+
		`(draw_id,user_id,edition_id,round_no,operation_id,mode,preview_id,pool_code,rule_version,requested_count,actual_count,unit_petal_cost,charged_petals,refunded_petals,consumed_petals,flowers_added,reward_snapshot_json,draw_seq,occurred_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		w.DrawID, s.User, s.Edition, w.Round, w.Operation, w.Mode, preview, w.Pool, w.Version, w.Requested, w.Actual, w.UnitCost, w.Charged, w.Refund, w.Consumed, w.FlowersAdded, w.Snapshot, w.Seq, w.Occurred); e != nil {
		return e
	}
	if e := BatchInsert(s.Ctx, s.Tx, platform.Shard("draw_detail", s.User),
		[]string{"detail_id", "user_id", "edition_id", "draw_id", "draw_no", "original_reward_code", "original_item_name", "quantity", "petal_cost", "flowers_before", "probability_lit", "guarantee_lit", "flowers_after", "cumulative_gold_equivalent"}, w.Details); e != nil {
		return e
	}
	if e := BatchInsert(s.Ctx, s.Tx, platform.Shard("flower_lit_log", s.User),
		[]string{"log_id", "user_id", "edition_id", "round_no", "draw_id", "draw_no", "pool_code", "flowers_added", "flowers_after"}, w.Flowers); e != nil {
		return e
	}
	if e := BatchInsert(s.Ctx, s.Tx, "reward_grant",
		[]string{"grant_id", "user_id", "edition_id", "round_no", "source_type", "source_id", "source_entry_no", "draw_id", "draw_no", "original_reward_code", "original_item_name", "quantity", "candidate_snapshot", "status", "final_item_id", "final_item_name"}, w.Grants); e != nil {
		return e
	}
	if _, e := s.Tx.Exec(s.Ctx, `UPDATE user_round SET flower_count=?,consumed_gold_equivalent=?,unlocked_chests=?,status=? WHERE user_id=? AND edition_id=? AND round_no=?`, w.FlowersAfter, w.GoldValue, w.Unlocked, w.Status, s.User, s.Edition, w.Round); e != nil {
		return e
	}
	if _, e := s.Tx.Exec(s.Ctx, `UPDATE user_participation SET formal_draw_count=formal_draw_count+1,draw_seq=?,draw_progress_version=draw_progress_version+1,version=version+1,ranking_score=?,ranking_score_reached_at_sec=? WHERE user_id=? AND edition_id=?`, w.Seq, s.Score, w.Occurred.Unix(), s.User, s.Edition); e != nil {
		return e
	}
	if w.Idempotent {
		if _, e := s.Tx.Exec(s.Ctx, `INSERT INTO idempotency_request(user_id,operation_type,operation_id,request_hash,accepted_at,recover_until,result_user_id,result_draw_id) VALUES(?,'DRAW',?,?,?,DATE_ADD(?,INTERVAL 24 HOUR),?,?)`, s.User, w.Operation, w.RequestHash[:], s.Now, s.Now, s.User, w.DrawID); e != nil {
			return e
		}
	}
	return nil
}
