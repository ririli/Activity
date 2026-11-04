package mysql

import (
	"activity/pkg/platform"
	lottery "activity/services/activity-service/internal/lottery/domain"
	"activity/services/activity-service/internal/participation/application"
	"activity/services/activity-service/internal/participation/domain"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"math"
	"strconv"
	"time"
)

type Repository struct {
	DB     *sql.DB
	Redis  *redis.Client
	Notify chan []byte
}
type state struct {
	messages                                                           []map[string]any
	nickname                                                           string
	tx                                                                 *sql.Tx
	ctx                                                                context.Context
	user, edition                                                      uint64
	round, version, flowers, unlocked, completed, mask                 int
	gold, petals, ledger, value, formal, seq, progress, score, reached int64
	rules                                                              domain.Rules
	now                                                                time.Time
	entry                                                              int
	biz                                                                uint64
}

func (r *Repository) Execute(ctx context.Context, u string, c application.Command) (any, error) {
	user, e := platform.UID(u)
	if e != nil {
		return nil, e
	}
	edition, e := platform.UID(c.Edition)
	if e != nil {
		return nil, e
	}
	if c.Action == "draw" {
		readTx, readErr := r.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if readErr != nil {
			return nil, readErr
		}
		readState := &state{tx: readTx, ctx: ctx, user: user}
		v, found, readErr := readState.recover(c)
		readTx.Rollback()
		if readErr != nil {
			return nil, readErr
		}
		if found {
			return v, nil
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()
	key := "activity:" + platform.Env("APP_ENV", "local") + ":user:" + u + ":mutation"
	token := platform.Random()
	ok, e := r.Redis.SetNX(ctx, key, token, 10*time.Second).Result()
	if e != nil {
		return nil, e
	}
	if !ok {
		return nil, platform.Fail(409, "OPERATION_IN_PROGRESS")
	}
	defer func() {
		release, c := context.WithTimeout(context.Background(), time.Second)
		defer c()
		r.Redis.Eval(release, `if redis.call('GET',KEYS[1])==ARGV[1] then return redis.call('DEL',KEYS[1]) end return 0`, []string{key}, token)
	}()
	tx, e := r.DB.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	s := &state{tx: tx, ctx: ctx, user: user, edition: edition, biz: platform.ID()}
	s.nickname = c.DisplayName
	if c.Action == "draw" {
		v, found, e := s.recover(c)
		if e != nil {
			return nil, e
		}
		if found {
			return v, nil
		}
	}
	active := c.Action != "settle" && c.Action != "select" && c.Action != "chest-confirm" && c.Action != "preview-abandon"
	if e = s.load(active); e != nil {
		return nil, e
	}
	var out any
	switch c.Action {
	case "settle":
		e = s.settleUser()
		out = map[string]string{"status": "COMPLETED"}
	case "join":
		out = s.summary()
	case "draw":
		out, e = s.draw(c, nil, 0)
	case "purchase":
		out, e = s.purchase(c)
	case "stage":
		out, e = s.stage(c.Stage, "USER")
	case "chest-open":
		out, e = s.chestOpen()
	case "chest-confirm":
		out, e = s.chestConfirm(c.ID)
	case "select":
		out, e = s.selectReward(c.ID, c.Item)
	case "next-round":
		out, e = s.nextRound(c.Round)
	case "preview":
		out, e = s.preview()
	case "preview-confirm":
		out, e = s.confirmPreview()
	case "preview-abandon":
		out, e = s.abandonPreview()
	default:
		e = platform.Fail(400, "UNKNOWN_ACTION")
	}
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, platform.Fail(503, "OPERATION_RESULT_UNKNOWN")
	}
	for _, m := range s.messages {
		select {
		case r.Notify <- platform.JSON(m):
		default:
		}
	}
	return out, nil
}
func (s *state) load(active bool) error {
	var status string
	var start, end time.Time
	var rv int
	e := s.tx.QueryRowContext(s.ctx, `SELECT status,starts_at,ends_at,active_rule_version,UTC_TIMESTAMP(6) FROM activity_edition WHERE edition_id=? FOR SHARE`, s.edition).Scan(&status, &start, &end, &rv, &s.now)
	if e != nil {
		return e
	}
	if active && (status != "ACTIVE" || s.now.Before(start) || !s.now.Before(end)) {
		return platform.Fail(409, "ACTIVITY_NOT_ACTIVE")
	}
	e = s.tx.QueryRowContext(s.ctx, `SELECT gold_balance,petal_balance,ledger_seq FROM wallet WHERE user_id=? FOR UPDATE NOWAIT`, s.user).Scan(&s.gold, &s.petals, &s.ledger)
	if errors.Is(e, sql.ErrNoRows) {
		return platform.Fail(409, "ACCOUNT_INITIALIZING")
	}
	if e != nil {
		return e
	}
	var exists int
	e = s.tx.QueryRowContext(s.ctx, `SELECT COUNT(*) FROM user_participation WHERE user_id=? AND edition_id=?`, s.user, s.edition).Scan(&exists)
	if e != nil {
		return e
	}
	if exists == 0 {
		if !active {
			return platform.Fail(409, "NOT_PARTICIPATING")
		}
		_, e = s.tx.ExecContext(s.ctx, `INSERT INTO user_participation(user_id,edition_id,ranking_score_reached_at_sec) VALUES(?,?,?)`, s.user, s.edition, s.now.Unix())
		if e != nil {
			return e
		}
		_, e = s.tx.ExecContext(s.ctx, `INSERT INTO user_round(user_id,edition_id,round_no,rule_version) VALUES(?,?,1,?)`, s.user, s.edition, rv)
		if e != nil {
			return e
		}
		e = platform.Outbox(s.ctx, s.tx, "ParticipationInitialized.v1", s.user, s.edition, map[string]string{"user_id": fmt.Sprint(s.user), "edition_id": fmt.Sprint(s.edition), "participation_created_at_sec": fmt.Sprint(s.now.Unix())})
		if e != nil {
			return e
		}
	}
	e = s.tx.QueryRowContext(s.ctx, `SELECT current_round_no,formal_draw_count,draw_seq,draw_progress_version,ranking_score,ranking_score_reached_at_sec FROM user_participation WHERE user_id=? AND edition_id=? FOR UPDATE`, s.user, s.edition).Scan(&s.round, &s.formal, &s.seq, &s.progress, &s.score, &s.reached)
	if e != nil {
		return e
	}
	e = s.tx.QueryRowContext(s.ctx, `SELECT rule_version,flower_count,consumed_gold_equivalent,unlocked_chests,completed_chests,claimed_stage_mask FROM user_round WHERE user_id=? AND edition_id=? AND round_no=? FOR UPDATE`, s.user, s.edition, s.round).Scan(&s.version, &s.flowers, &s.value, &s.unlocked, &s.completed, &s.mask)
	if e != nil {
		return e
	}
	var rules []byte
	var hash []byte
	e = s.tx.QueryRowContext(s.ctx, `SELECT rule_json,checksum FROM activity_rule_version WHERE edition_id=? AND version=?`, s.edition, s.version).Scan(&rules, &hash)
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
	return json.Unmarshal(rules, &s.rules)
}
func (s *state) summary() any {
	return map[string]any{"edition_id": fmt.Sprint(s.edition), "gold": s.gold, "petals": s.petals, "round_no": s.round, "flowers": s.flowers, "unlocked_chests": s.unlocked, "completed_chests": s.completed, "claimed_stage_mask": s.mask, "score": s.score, "rules": s.rules}
}
func (s *state) asset(asset string, delta int64, reason, kind string) error {
	if delta == 0 {
		return nil
	}
	balance := s.petals
	col := "petal_balance"
	if asset == "GOLD" {
		balance = s.gold
		col = "gold_balance"
	}
	if delta > 0 && balance > math.MaxInt64-delta || delta < 0 && balance < -delta {
		return platform.Fail(409, "INSUFFICIENT_OR_OVERFLOWING_ASSET")
	}
	balance += delta
	result, e := s.tx.ExecContext(s.ctx, fmt.Sprintf(`UPDATE wallet SET %s=%s+?,ledger_seq=ledger_seq+1,version=version+1 WHERE user_id=? AND %s>=?`, col, col, col), delta, s.user, max(0, -delta))
	if e != nil {
		return e
	}
	n, e := result.RowsAffected()
	if e != nil || n != 1 {
		return platform.Fail(409, "ASSET_CONFLICT")
	}
	s.ledger++
	s.entry++
	_, e = s.tx.ExecContext(s.ctx, fmt.Sprintf(`INSERT INTO %s(ledger_id,user_id,edition_id,asset_type,delta,balance_after,reason,biz_type,biz_id,entry_no,user_ledger_seq) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, platform.Shard("wallet_ledger", s.user)), platform.ID(), s.user, s.edition, asset, delta, balance, reason, kind, s.biz, s.entry, s.ledger)
	if asset == "GOLD" {
		s.gold = balance
	} else {
		s.petals = balance
	}
	return e
}
func (s *state) grant(p domain.Prize, source string, sourceID uint64, no int, draw any) (uint64, error) {
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
	_, e := s.tx.ExecContext(s.ctx, `INSERT INTO reward_grant(grant_id,user_id,edition_id,round_no,source_type,source_id,source_entry_no,draw_id,draw_no,original_reward_code,original_item_name,quantity,candidate_snapshot,status,final_item_id,final_item_name) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, id, s.user, s.edition, s.round, source, sourceID, no, draw, drawNo, p.Code, p.Name, p.Quantity, candidates, status, finalID, finalName)
	return id, e
}
func (s *state) recover(c application.Command) (any, bool, error) {
	var hash []byte
	var until time.Time
	var draw uint64
	e := s.tx.QueryRowContext(s.ctx, `SELECT request_hash,recover_until,result_draw_id FROM idempotency_request WHERE user_id=? AND operation_type='DRAW' AND operation_id=?`, s.user, c.Operation).Scan(&hash, &until, &draw)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, false, nil
	}
	if e != nil {
		return nil, false, e
	}
	expected := requestHash(c)
	if string(hash) != string(expected[:]) {
		return nil, false, platform.Fail(409, "IDEMPOTENCY_CONFLICT")
	}
	if !until.After(time.Now()) {
		return nil, false, platform.Fail(409, "IDEMPOTENCY_EXPIRED")
	}
	var raw []byte
	var cost, refund int64
	var requested, actual, flowers int
	e = s.tx.QueryRowContext(s.ctx, fmt.Sprintf(`SELECT reward_snapshot_json,consumed_petals,refunded_petals,requested_count,actual_count FROM %s WHERE user_id=? AND draw_id=?`, platform.Shard("draw_order", s.user)), s.user, draw).Scan(&raw, &cost, &refund, &requested, &actual)
	var steps []domain.Step
	if e == nil {
		e = json.Unmarshal(raw, &steps)
		if len(steps) > 0 {
			flowers = steps[len(steps)-1].After
		}
	}
	return map[string]any{"draw_id": fmt.Sprint(draw), "results": json.RawMessage(raw), "consumed_petals": cost, "refunded_petals": refund, "requested_count": requested, "actual_count": actual, "flowers": flowers}, true, e
}
func requestHash(c application.Command) [32]byte {
	return sha256.Sum256(platform.JSON([]any{c.Edition, c.Pool, c.Count, "NORMAL"}))
}
func (s *state) draw(c application.Command, fixed *domain.Plan, previewID uint64) (any, error) {
	pool, ok := s.rules.Pools[c.Pool]
	if !ok {
		return nil, platform.Fail(400, "INVALID_POOL")
	}
	if s.flowers >= 18 {
		return nil, platform.Fail(409, "ROUND_FULL")
	}
	if fixed == nil && (pool.Cost <= 0 || pool.Cost > math.MaxInt64/int64(c.Count) || s.petals < pool.Cost*int64(c.Count)) {
		return nil, platform.Fail(409, "INSUFFICIENT_PETALS")
	}
	var plan domain.Plan
	var e error
	if fixed == nil {
		plan, e = domain.Simulate(pool, lottery.Progress{Flowers: s.flowers, GoldValue: s.value}, c.Count, domain.Random)
		if e != nil {
			return nil, e
		}
	} else {
		plan = *fixed
	}
	before := s.flowers
	mode := "NORMAL"
	source := c.Pool + "_DRAW"
	charged := plan.Cost + plan.Refund
	var pid any
	if previewID != 0 {
		pid = previewID
		mode = "PREVIEW"
		source = "PREVIEW"
		charged = plan.Cost
		plan.Refund = 0
	}
	if e = s.asset("PETAL", -charged, "DRAW_CHARGE", "DRAW"); e != nil {
		return nil, e
	}
	var gold, petals int64
	for _, step := range plan.Steps {
		if step.Prize.ID == 1001 {
			if step.Prize.Quantity > math.MaxInt64-gold {
				return nil, errors.New("prize overflow")
			}
			gold += step.Prize.Quantity
		}
		if step.Prize.ID == 1002 {
			if step.Prize.Quantity > math.MaxInt64-petals {
				return nil, errors.New("prize overflow")
			}
			petals += step.Prize.Quantity
		}
	}
	if e = s.asset("GOLD", gold, "DRAW_WIN", "DRAW"); e != nil {
		return nil, e
	}
	if e = s.asset("PETAL", petals, "DRAW_WIN", "DRAW"); e != nil {
		return nil, e
	}
	if e = s.asset("PETAL", plan.Refund, "DRAW_REFUND", "DRAW"); e != nil {
		return nil, e
	}
	if plan.Cost > math.MaxInt64-s.score {
		return nil, errors.New("score overflow")
	}
	s.seq++
	s.score += plan.Cost
	occurred := max(s.now.Unix(), s.reached)
	_, e = s.tx.ExecContext(s.ctx, fmt.Sprintf(`INSERT INTO %s(draw_id,user_id,edition_id,round_no,operation_id,mode,preview_id,pool_code,rule_version,requested_count,actual_count,unit_petal_cost,charged_petals,refunded_petals,consumed_petals,flowers_added,reward_snapshot_json,draw_seq,occurred_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, platform.Shard("draw_order", s.user)), s.biz, s.user, s.edition, s.round, c.Operation, mode, pid, c.Pool, s.version, c.Count, len(plan.Steps), pool.Cost, charged, plan.Refund, plan.Cost, plan.Flowers-before, platform.JSON(plan.Steps), s.seq, time.Unix(occurred, 0).UTC())
	if e != nil {
		return nil, e
	}
	for i, step := range plan.Steps {
		no := i + 1
		_, e = s.tx.ExecContext(s.ctx, fmt.Sprintf(`INSERT INTO %s(detail_id,user_id,edition_id,draw_id,draw_no,original_reward_code,original_item_name,quantity,petal_cost,flowers_before,probability_lit,guarantee_lit,flowers_after,cumulative_gold_equivalent) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, platform.Shard("draw_detail", s.user)), platform.ID(), s.user, s.edition, s.biz, no, step.Prize.Code, step.Prize.Name, step.Prize.Quantity, pool.Cost, step.Before, step.Probability, step.After-step.Before-step.Probability, step.After, step.GoldValue)
		if e != nil {
			return nil, e
		}
		if step.After > step.Before {
			_, e = s.tx.ExecContext(s.ctx, fmt.Sprintf(`INSERT INTO %s(log_id,user_id,edition_id,round_no,draw_id,draw_no,pool_code,flowers_added,flowers_after) VALUES(?,?,?,?,?,?,?,?,?)`, platform.Shard("flower_lit_log", s.user)), platform.ID(), s.user, s.edition, s.round, s.biz, no, c.Pool, step.After-step.Before, step.After)
			if e != nil {
				return nil, e
			}
		}
		grantID, grantErr := s.grant(step.Prize, source, s.biz, no, s.biz)
		if grantErr == nil && source != "PREVIEW" {
			s.notify(step.Prize, grantID, "normal_draw")
		}
		if e = grantErr; e != nil {
			return nil, e
		}
	}
	status := "ACTIVE"
	if plan.Flowers == 18 {
		status = "FULL"
	}
	_, e = s.tx.ExecContext(s.ctx, `UPDATE user_round SET flower_count=?,consumed_gold_equivalent=?,unlocked_chests=?,status=? WHERE user_id=? AND edition_id=? AND round_no=?`, plan.Flowers, plan.GoldValue, plan.Flowers/6, status, s.user, s.edition, s.round)
	if e != nil {
		return nil, e
	}
	_, e = s.tx.ExecContext(s.ctx, `UPDATE user_participation SET formal_draw_count=formal_draw_count+1,draw_seq=?,draw_progress_version=draw_progress_version+1,version=version+1,ranking_score=?,ranking_score_reached_at_sec=? WHERE user_id=? AND edition_id=?`, s.seq, s.score, occurred, s.user, s.edition)
	if e != nil {
		return nil, e
	}
	if previewID == 0 {
		h := requestHash(c)
		_, e = s.tx.ExecContext(s.ctx, `INSERT INTO idempotency_request(user_id,operation_type,operation_id,request_hash,accepted_at,recover_until,result_user_id,result_draw_id) VALUES(?,'DRAW',?,?,?,DATE_ADD(?,INTERVAL 24 HOUR),?,?)`, s.user, c.Operation, h[:], s.now, s.now, s.user, s.biz)
		if e != nil {
			return nil, e
		}
	}
	e = platform.Outbox(s.ctx, s.tx, "FormalDrawCommitted.v1", s.user, s.biz, map[string]string{"user_id": fmt.Sprint(s.user), "edition_id": fmt.Sprint(s.edition), "draw_id": fmt.Sprint(s.biz), "draw_seq": fmt.Sprint(s.seq), "consumed_petals": fmt.Sprint(plan.Cost), "occurred_at_sec": fmt.Sprint(occurred), "total_score": fmt.Sprint(s.score), "score_reached_at_sec": fmt.Sprint(occurred)})
	return map[string]any{"draw_id": fmt.Sprint(s.biz), "results": plan.Steps, "consumed_petals": plan.Cost, "refunded_petals": plan.Refund, "requested_count": c.Count, "actual_count": len(plan.Steps), "flowers": plan.Flowers}, e
}
func (r *Repository) Initialize(ctx context.Context, ev platform.Event) error {
	if ev.Type != "AccountCreated.v1" {
		return platform.Fail(409, "INVALID_EVENT_TYPE")
	}
	var p struct {
		User string `json:"user_id"`
	}
	if e := json.Unmarshal(ev.Payload, &p); e != nil {
		return platform.Fail(409, "INVALID_EVENT")
	}
	user, e := platform.UID(p.User)
	if e != nil {
		return platform.Fail(409, "INVALID_EVENT_USER")
	}
	tx, e := r.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	res, e := tx.ExecContext(ctx, `INSERT IGNORE INTO inbox_event(consumer_name,event_id) VALUES('account-init',?)`, ev.ID)
	if e != nil {
		return e
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return tx.Commit()
	}
	res, e = tx.ExecContext(ctx, `INSERT IGNORE INTO wallet(user_id,gold_balance,petal_balance,ledger_seq) VALUES(?,10000000,100000000,2)`, user)
	if e != nil {
		return e
	}
	n, _ = res.RowsAffected()
	if n == 1 {
		for i, a := range []string{"GOLD", "PETAL"} {
			v := int64(10000000)
			if i == 1 {
				v = 100000000
			}
			_, e = tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s(ledger_id,user_id,asset_type,delta,balance_after,reason,biz_type,biz_id,entry_no,user_ledger_seq) VALUES(?,?,?,?,?,'INITIALIZATION','ACCOUNT',?,?,?)`, platform.Shard("wallet_ledger", user)), platform.ID(), user, a, v, v, user, i+1, i+1)
			if e != nil {
				return e
			}
		}
	}
	return tx.Commit()
}
func parseID(v string) (uint64, error) { return strconv.ParseUint(v, 10, 63) }
