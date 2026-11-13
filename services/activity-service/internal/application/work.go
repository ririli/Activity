package application

import (
	assetsapp "activity/services/activity-service/internal/assets/application"
	assetsdomain "activity/services/activity-service/internal/assets/domain"
	"activity/services/activity-service/internal/fault"
	lotteryapp "activity/services/activity-service/internal/lottery/application"
	lottery "activity/services/activity-service/internal/lottery/domain"
	partapp "activity/services/activity-service/internal/participation/application"
	partdomain "activity/services/activity-service/internal/participation/domain"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// work belongs only to the service-level orchestration. Modules receive their
// own domain objects/commands, never this orchestration object.
type work struct {
	session       Session
	assets        assetsapp.Service
	lottery       lotteryapp.Service
	participation partapp.Service
	wallet        assetsdomain.Wallet
	progress      partdomain.Progress
	edition       partdomain.Edition
	rules         Rules
	scope         Scope
	now           time.Time
	nickname      string
	runtime       Runtime
	messages      []map[string]any
}

func encode(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return raw
}
func (s *work) Load(active, scoring bool) error {
	gate, err := s.session.Participation.Gate()
	if err != nil {
		return err
	}
	s.edition = gate
	if active {
		if err = gate.CheckActive(); err != nil {
			return err
		}
	}
	s.now = gate.Now
	s.wallet, err = s.session.Assets.LoadWallet()
	if errors.Is(err, fault.ErrNotFound) {
		return fault.New("ACCOUNT_INITIALIZING")
	}
	if err != nil {
		return err
	}
	exists, err := s.session.Participation.Exists()
	if err != nil {
		return err
	}
	if active && gate.HourlyRanking && (!exists || scoring) {
		s.now, err = s.session.AdmitHourly(gate.End)
		if err != nil {
			return err
		}
		gate.Now = s.now
		if err := gate.CheckActive(); err != nil {
			return err
		}
	}
	if !exists {
		if !active {
			return fault.New("NOT_PARTICIPATING")
		}
		if err = s.session.Participation.Initialize(gate.RuleVersion, s.now.Unix()); err != nil {
			return err
		}
		if err = s.Reliable("ParticipationInitialized.v1", s.scope.User, s.scope.Edition, map[string]string{"user_id": fmt.Sprint(s.scope.User), "edition_id": fmt.Sprint(s.scope.Edition), "participation_created_at_sec": fmt.Sprint(s.now.Unix())}); err != nil {
			return err
		}
	}
	s.progress, err = s.session.Participation.LoadProgress()
	if err != nil {
		return err
	}
	s.session.BindRound(s.progress.Round, s.progress.Progress)
	raw, hash, err := s.session.Participation.Rule(s.progress.Version)
	if err != nil {
		return err
	}
	actual, err := lotteryapp.CanonicalHash(raw)
	if err != nil {
		return err
	}
	if string(actual[:]) != string(hash) {
		return errors.New("rule checksum mismatch")
	}
	return json.Unmarshal(raw, &s.rules)
}
func (s *work) Summary() any {
	return map[string]any{"edition_id": fmt.Sprint(s.scope.Edition), "gold": s.wallet.Gold, "petals": s.wallet.Petals, "round_no": s.progress.Round, "flowers": s.progress.Flowers, "unlocked_chests": s.progress.Unlocked, "completed_chests": s.progress.Completed, "claimed_stage_mask": s.progress.Mask, "score": s.progress.Score, "rules": s.rules}
}
func (s *work) Asset(asset string, delta int64, reason, kind string) error {
	return s.assets.Apply(&s.wallet, asset, delta, reason, kind)
}
func assetPrize(p lottery.Prize) assetsdomain.Prize {
	out := assetsdomain.Prize{Code: p.Code, ID: p.ID, Name: p.Name, Quantity: p.Quantity, Weight: p.Weight}
	for _, c := range p.Choices {
		out.Choices = append(out.Choices, assetPrize(c))
	}
	return out
}
func (s *work) Grant(p lottery.Prize, source string, sourceID uint64, no int) (uint64, error) {
	return s.assets.Grant(assetPrize(p), s.progress.Round, source, sourceID, no, 0)
}
func (s *work) Reliable(kind string, user, aggregate uint64, payload any) error {
	ev := Event{ID: fmt.Sprint(s.runtime.NewID()), Type: kind, Payload: encode(payload)}
	return s.session.Outbox.Append(ev, user, aggregate)
}
func (s *work) Notify(p lottery.Prize, grant uint64, source string) {
	if s.nickname == "" || len(p.Choices) > 0 || p.ID == 1001 || p.ID == 1002 {
		return
	}
	s.messages = append(s.messages, map[string]any{"schema_version": 1, "event_id": fmt.Sprintf("ring:%d:%s:v1", grant, source), "event_type": "ring_awarded", "occurred_at": s.now.UTC(), "expires_at": s.now.Add(30 * time.Second).UTC(), "edition_id": fmt.Sprint(s.scope.Edition), "source": source, "display_name": s.nickname, "item_id": p.ID, "item_name": p.Name, "quantity": p.Quantity})
}
func (s *work) Recover(c lotteryapp.DrawCommand) (any, bool, error) {
	return s.lottery.Recover(c)
}
