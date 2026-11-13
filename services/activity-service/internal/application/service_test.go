package application

import (
	assets "activity/services/activity-service/internal/assets/application"
	ad "activity/services/activity-service/internal/assets/domain"
	"activity/services/activity-service/internal/fault"
	lottery "activity/services/activity-service/internal/lottery/application"
	ld "activity/services/activity-service/internal/lottery/domain"
	part "activity/services/activity-service/internal/participation/application"
	pd "activity/services/activity-service/internal/participation/domain"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"
)

// These transaction fakes exercise the public use cases with independent module
// ports. SQL connection/commit behavior is tested separately in infrastructure.
type memory struct {
	Wallet          ad.Wallet
	Progress        pd.Progress
	Changes         []assets.Change
	Grants          []assets.Grant
	Draws           []lottery.DrawWrite
	Events          []Event
	Preview         ld.Preview
	TopUps          int
	Recovered       map[string]lottery.Recovery
	Ended           bool
	Reward          ad.RewardGrant
	PurchaseEdition uint64
}
type memoryUoW struct {
	state         memory
	fail          string
	commitUnknown bool
	writes, reads int
	now           time.Time
	randoms       int
}
type assetRepo struct {
	assets.Repository
	u *memoryUoW
	m *memory
}
type lotteryRepo struct {
	lottery.Repository
	u *memoryUoW
	m *memory
}
type partRepo struct {
	part.Repository
	u *memoryUoW
	m *memory
}
type outboxRepo struct {
	u *memoryUoW
	m *memory
}

func (u *memoryUoW) check(stage string) error {
	if u.fail == stage {
		return fmt.Errorf("injected %s failure", stage)
	}
	return nil
}
func (a assetRepo) LoadWallet() (ad.Wallet, error) { return a.m.Wallet, nil }
func (a assetRepo) ApplyChange(c assets.Change) error {
	if err := a.u.check(c.Reason); err != nil {
		return err
	}
	if c.Asset == ad.Gold {
		a.m.Wallet.Gold = c.Balance
	} else {
		a.m.Wallet.Petals = c.Balance
	}
	a.m.Wallet.LedgerSeq = c.Ledger
	a.m.Changes = append(a.m.Changes, c)
	return nil
}
func (a assetRepo) SaveGrants(g []assets.Grant) error {
	if err := a.u.check("grants"); err != nil {
		return err
	}
	a.m.Grants = append(a.m.Grants, g...)
	return nil
}
func (a assetRepo) InsertPreviewTopUpFor(_, _ uint64, _ string, _, _ int64, _ int) error {
	a.m.TopUps++
	return nil
}
func (l lotteryRepo) FindDraw(op string) (lottery.Recovery, error) {
	if v, ok := l.m.Recovered[op]; ok {
		return v, nil
	}
	return lottery.Recovery{}, fault.ErrNotFound
}
func (l lotteryRepo) SaveDraw(w lottery.DrawWrite) error {
	if err := l.u.check("draw"); err != nil {
		return err
	}
	l.m.Draws = append(l.m.Draws, w)
	if w.Idempotent {
		hash := w.RequestHash
		result := lottery.DrawResult{DrawID: fmt.Sprint(w.ID), Consumed: w.Plan.Cost, Refunded: w.Plan.Refund, Requested: w.Requested, Actual: len(w.Plan.Steps), Flowers: w.Plan.Flowers, Results: []lottery.DrawStep{}}
		for i, step := range w.Plan.Steps {
			result.Results = append(result.Results, lottery.DrawStep{Step: step, GrantID: fmt.Sprint(l.m.Grants[i].ID)})
		}
		l.m.Recovered[w.Operation] = lottery.Recovery{Hash: hash[:], Until: w.Accepted.Add(24 * time.Hour), Result: result}
	}
	return nil
}
func (l lotteryRepo) LockPreview() (ld.Preview, error) { return l.m.Preview, nil }
func (l lotteryRepo) FindPreview() (uint64, []byte, string, int64, error) {
	if l.m.Preview.ID == 0 {
		return 0, nil, "", 0, fault.ErrNotFound
	}
	var p ld.Plan
	if err := json.Unmarshal(l.m.Preview.Expected, &p); err != nil {
		return 0, nil, "", 0, err
	}
	return l.m.Preview.ID, l.m.Preview.Raw, l.m.Preview.Status, p.Cost, nil
}
func (l lotteryRepo) InsertPreview(id uint64, round, version, actual int, cost int64, snapshot, raw, expected, hash []byte) error {
	l.m.Preview = ld.Preview{ID: id, Round: round, Version: l.m.Progress.Progress, Status: "PENDING", Raw: raw, Expected: expected, Hash: hash}
	return nil
}
func (l lotteryRepo) LoadDraw(id uint64) (lottery.DrawResult, error) {
	for _, d := range l.m.Draws {
		if d.ID != id {
			continue
		}
		v := lottery.DrawResult{DrawID: fmt.Sprint(id), Consumed: d.Plan.Cost, Refunded: d.Plan.Refund, Requested: d.Requested, Actual: len(d.Plan.Steps), Flowers: d.Plan.Flowers}
		for i, step := range d.Plan.Steps {
			v.Results = append(v.Results, lottery.DrawStep{Step: step, GrantID: fmt.Sprint(l.m.Grants[i].ID)})
		}
		return v, nil
	}
	return lottery.DrawResult{}, fault.ErrNotFound
}
func (a assetRepo) FindPurchase(string) (uint64, uint64, string, string, int64, error) {
	return 9, a.m.PurchaseEdition, "SHOP", "52", 52, nil
}
func (l lotteryRepo) ConfirmPreview(_, draw uint64) error {
	if err := l.u.check("preview-confirm"); err != nil {
		return err
	}
	l.m.Preview.Status = "CONFIRMED"
	l.m.Preview.DrawID = draw
	return nil
}
func (p partRepo) Gate() (pd.Edition, error) {
	status := "ACTIVE"
	if p.m.Ended {
		status = "ENDED"
	}
	return pd.Edition{Status: status, Start: p.u.now.Add(-time.Hour), End: p.u.now.Add(time.Hour), Now: p.u.now, RuleVersion: 1}, nil
}
func (p partRepo) Exists() (bool, error)              { return true, nil }
func (p partRepo) LoadProgress() (pd.Progress, error) { return p.m.Progress, nil }
func (p partRepo) Rule(int) ([]byte, []byte, error) {
	raw := encode(DefaultRules())
	hash, err := lottery.CanonicalHash(raw)
	return raw, hash[:], err
}
func (p partRepo) SaveDrawProgress(v pd.Progress) error {
	if err := p.u.check("progress"); err != nil {
		return err
	}
	p.m.Progress = v
	return nil
}
func (o outboxRepo) Append(e Event, _, _ uint64) error {
	if err := o.u.check("outbox"); err != nil {
		return err
	}
	o.m.Events = append(o.m.Events, e)
	return nil
}
func (u *memoryUoW) session(m *memory) Session {
	return Session{Assets: assetRepo{u: u, m: m}, Lottery: lotteryRepo{u: u, m: m}, Participation: partRepo{u: u, m: m}, Outbox: outboxRepo{u, m}, BindRound: func(int, int64) {}}
}
func (u *memoryUoW) Within(ctx context.Context, _ Scope, fn func(Session) error) error {
	u.writes++
	next := u.state
	next.Recovered = map[string]lottery.Recovery{}
	for k, v := range u.state.Recovered {
		next.Recovered[k] = v
	}
	if err := fn(u.session(&next)); err != nil {
		return err
	}
	u.state = next
	if u.commitUnknown {
		return fault.ErrCommitUnknown
	}
	return nil
}
func (u *memoryUoW) Read(ctx context.Context, _ Scope, fn func(Session) error) error {
	u.reads++
	return fn(u.session(&u.state))
}

type lockStub struct{ calls int }

func (l *lockStub) Acquire(context.Context, string) (func(), error) { l.calls++; return func() {}, nil }

type effectsStub struct {
	messages []map[string]any
}

func (e *effectsStub) Enqueue(v []map[string]any) { e.messages = append(e.messages, v...) }
func fixture() (Service, *memoryUoW, *effectsStub, *lockStub) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	u := &memoryUoW{now: now, state: memory{Wallet: ad.Wallet{Gold: 10000, Petals: 1000}, Progress: pd.Progress{Round: 1, Version: 1}, Recovered: map[string]lottery.Recovery{}}}
	effects := &effectsStub{}
	lock := &lockStub{}
	id := uint64(100)
	s := Service{UoW: u, Lock: lock, Effects: effects, Runtime: Runtime{NewID: func() uint64 { id++; return id }, Now: func() time.Time { return now }, Random: func(n int64) (int64, error) { u.randoms++; return 0, nil }}}
	return s, u, effects, lock
}
func drawCommand() DrawCommand {
	return DrawCommand{Actor: Actor{User: "7", Edition: "8", DisplayName: "tester"}, Operation: "draw-1", Pool: "NIGHT", Count: 1}
}
func TestDrawCoordinatesThreeModules(t *testing.T) {
	s, u, _, _ := fixture()
	out, err := s.Draw(context.Background(), drawCommand())
	if err != nil {
		t.Fatal(err)
	}
	result := out.(lottery.DrawResult)
	if result.Actual != 1 || result.Flowers != 3 || result.Consumed != 100 {
		t.Fatalf("result %+v", result)
	}
	if u.state.Wallet.Petals != 900 || u.state.Wallet.Gold != 11000 || u.state.Progress.Score != 100 || len(u.state.Grants) != 1 || len(u.state.Draws) != 1 || len(u.state.Events) != 1 {
		t.Fatalf("incomplete transaction: %+v", u.state)
	}
	if !reflect.DeepEqual(result.Results[0].Step, u.state.Draws[0].Plan.Steps[0]) {
		t.Fatal("response and persisted snapshot diverged")
	}
}
func TestDrawFailuresRollbackAllModules(t *testing.T) {
	for _, stage := range []string{"DRAW_CHARGE", "DRAW_WIN", "grants", "progress", "draw", "outbox"} {
		t.Run(stage, func(t *testing.T) {
			s, u, e, _ := fixture()
			before := u.state
			u.fail = stage
			if _, err := s.Draw(context.Background(), drawCommand()); err == nil {
				t.Fatal("expected injected failure")
			}
			if !reflect.DeepEqual(u.state, before) {
				t.Fatalf("partial business state committed at %s", stage)
			}
			if len(e.messages) > 0 {
				t.Fatal("published effects after rollback")
			}
		})
	}
}
func TestFullRoundRefundAndImmutableResult(t *testing.T) {
	s, u, _, _ := fixture()
	u.state.Wallet.Petals = 150
	u.state.Progress.Flowers = 17
	u.state.Progress.Value = 2082200
	s.Runtime.Random = func(n int64) (int64, error) { return n - 1, nil }
	c := drawCommand()
	c.Pool = "DAY"
	c.Count = 30
	out, err := s.Draw(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	v := out.(lottery.DrawResult)
	if v.Actual != 1 || v.Consumed != 5 || v.Refunded != 145 || u.state.Wallet.Petals != 145 || u.state.Progress.Score != 5 {
		t.Fatalf("incorrect refund %+v", v)
	}
	if len(u.state.Changes) != 2 || u.state.Changes[0].Delta != -150 || u.state.Changes[1].Delta != 145 {
		t.Fatalf("classification lost %+v", u.state.Changes)
	}
}
func TestReplayBypassesCurrentActivityAndLock(t *testing.T) {
	s, u, _, l := fixture()
	c := drawCommand()
	first, err := s.Draw(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	before := u.state
	u.state.Ended = true
	u.state.Wallet.Petals = 0
	calls, randoms := l.calls, u.randoms
	second, err := s.Draw(context.Background(), c)
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatalf("replay %v %v", second, err)
	}
	if l.calls != calls || u.randoms != randoms || len(u.state.Draws) != len(before.Draws) {
		t.Fatal("replay executed a fresh operation")
	}
	c.Count = 10
	if _, err = s.Draw(context.Background(), c); err == nil || err.Error() != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("conflict %v", err)
	}
}
func TestExpiredReplayNeverExecutesAgain(t *testing.T) {
	s, u, _, l := fixture()
	c := drawCommand()
	if _, err := s.Draw(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	recovery := u.state.Recovered[c.Operation]
	recovery.Until = u.now
	u.state.Recovered[c.Operation] = recovery
	calls := l.calls
	if _, err := s.Draw(context.Background(), c); err == nil || err.Error() != "IDEMPOTENCY_EXPIRED" {
		t.Fatalf("expiry %v", err)
	}
	if len(u.state.Draws) != 1 || l.calls != calls {
		t.Fatal("expired operation re-executed")
	}
}
func TestUnknownCommitPublishesNothingAndCanRecover(t *testing.T) {
	s, u, e, _ := fixture()
	u.commitUnknown = true
	c := drawCommand()
	if _, err := s.Draw(context.Background(), c); err == nil || err.Error() != "OPERATION_RESULT_UNKNOWN" {
		t.Fatalf("commit outcome %v", err)
	}
	if len(e.messages) != 0 {
		t.Fatal("effects published before known commit")
	}
	if _, err := s.Draw(context.Background(), c); err != nil {
		t.Fatalf("recovery %v", err)
	}
	if len(u.state.Draws) != 1 {
		t.Fatal("duplicate draw")
	}
}
func TestPreviewConfirmationUsesSavedPlanWithoutRandomness(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			s, u, e, _ := fixture()
			u.state.Wallet.Petals = 0
			plan, err := ld.Simulate(DefaultRules().Pools["DAY"], ld.Progress{}, 1, func(int64) (int64, error) { return 0, nil })
			if err != nil {
				t.Fatal(err)
			}
			plan.Refund = 0
			raw := encode(plan.Steps)
			hash, _ := lottery.CanonicalHash(raw)
			u.state.Preview = ld.Preview{ID: 9, Round: 1, Version: 0, Status: "PENDING", Raw: raw, Expected: encode(plan), Hash: hash[:]}
			before := u.state
			if fail {
				u.fail = "preview-confirm"
			}
			out, err := s.ConfirmPreview(context.Background(), PreviewCommand{Actor: Actor{User: "7", Edition: "8"}})
			if fail {
				if err == nil || !reflect.DeepEqual(before, u.state) || len(e.messages) != 0 {
					t.Fatal("preview confirmation was partially committed")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			v := out.(lottery.DrawResult)
			if u.randoms != 0 || v.Requested != 180 || v.Actual != 1 || v.Refunded != 0 || u.state.TopUps != 1 || u.state.Preview.Status != "CONFIRMED" || u.state.Wallet.Gold != 9800 || u.state.Wallet.Petals != 0 {
				t.Fatalf("preview result %+v wallet %+v", v, u.state.Wallet)
			}
		})
	}
}
func TestBalanceCheckPrecedesWinningAssets(t *testing.T) {
	s, u, _, _ := fixture()
	u.state.Wallet.Petals = 99
	if _, err := s.Draw(context.Background(), drawCommand()); err == nil || err.Error() != "INSUFFICIENT_PETALS" {
		t.Fatalf("expected full request balance check: %v", err)
	}
	if u.randoms != 0 || len(u.state.Changes) != 0 {
		t.Fatal("simulated or charged an unaffordable draw")
	}
}

func (a assetRepo) LockReward(string) (assets.GrantState, error) {
	return assets.GrantState{Reward: a.m.Reward}, nil
}
func (a assetRepo) MarkRewardSelected(_ string, p ad.Prize) error {
	a.m.Reward.Status = "SELECTED"
	a.m.Reward.SelectedID = fmt.Sprint(p.ID)
	a.m.Reward.SelectedName = p.Name
	return nil
}
func (a assetRepo) RewardSource(string) (string, uint64, error) { return "DAY_DRAW", 9, nil }
func TestSelectionPublishesAuthenticatedNicknameOnlyOnce(t *testing.T) {
	s, u, e, _ := fixture()
	u.state.Reward = ad.RewardGrant{Status: "PENDING_CHOICE", Quantity: 1, Candidates: []ad.Prize{{ID: 3, Name: "ring", Quantity: 1}}}
	c := SelectRewardCommand{Actor: Actor{User: "7", Edition: "8", DisplayName: "tester"}, ID: "9", Item: "3"}
	if _, err := s.SelectReward(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if len(e.messages) != 1 || e.messages[0]["display_name"] != "tester" || e.messages[0]["source"] != "normal_draw_selection" {
		t.Fatalf("selection notification lost %+v", e.messages)
	}
	if _, err := s.SelectReward(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if len(e.messages) != 1 {
		t.Fatal("replayed selection published twice")
	}
}

func TestPreviewRetriesReturnCompleteOriginalResult(t *testing.T) {
	s, u, _, _ := fixture()
	c := PreviewCommand{Actor: Actor{User: "7", Edition: "8"}}
	first, err := s.Preview(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	randoms := u.randoms
	again, err := s.Preview(context.Background(), c)
	if err != nil || string(encode(first)) != string(encode(again)) || u.randoms != randoms {
		t.Fatalf("preview replay differs: %v / %v / %v", first, again, err)
	}
	confirmed, err := s.ConfirmPreview(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := s.ConfirmPreview(context.Background(), c)
	if err != nil || !reflect.DeepEqual(confirmed, replayed) || len(u.state.Draws) != 1 {
		t.Fatalf("confirmation replay differs: %v / %v / %v", confirmed, replayed, err)
	}
}

func TestPreviewCannotBeAbandonedAfterEditionEnds(t *testing.T) {
	s, u, _, _ := fixture()
	u.state.Ended = true
	_, err := s.AbandonPreview(context.Background(), PreviewCommand{Actor: Actor{User: "7", Edition: "8"}})
	if err == nil || err.Error() != "ACTIVITY_NOT_ACTIVE" {
		t.Fatalf("%v", err)
	}
}

func TestUserSettlementCannotRunBeforeEnd(t *testing.T) {
	s, _, _, _ := fixture()
	_, err := s.SettleUser(context.Background(), JoinCommand{Actor: Actor{User: "7", Edition: "8"}})
	if err == nil || err.Error() != "ACTIVITY_NOT_ENDED" {
		t.Fatalf("%v", err)
	}
}

func TestPurchaseOperationCannotCrossEditions(t *testing.T) {
	s, u, _, _ := fixture()
	u.state.PurchaseEdition = 9
	_, err := s.Purchase(context.Background(), PurchaseCommand{Actor: Actor{User: "7", Edition: "8"}, Operation: "buy-1", Product: "52"})
	if err == nil || err.Error() != "PURCHASE_CONFLICT" || len(u.state.Changes) != 0 {
		t.Fatalf("%v", err)
	}
}
