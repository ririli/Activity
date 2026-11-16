package application

import (
	domain "activity/services/activity-service/internal/assets/domain"
	"activity/services/activity-service/internal/fault"
	"math"
)

type Service struct {
	Repo  Repository
	NewID func() (uint64, error)
	Entry int
}

func (s *Service) Apply(w *domain.Wallet, asset string, delta int64, reason, kind string) error {
	if delta == 0 {
		return nil
	}
	if w.LedgerSeq == math.MaxInt64 {
		return fault.New("INSUFFICIENT_OR_OVERFLOWING_ASSET")
	}
	next := *w
	balance, err := next.Apply(asset, delta)
	if err != nil {
		return fault.New("INSUFFICIENT_OR_OVERFLOWING_ASSET")
	}
	next.LedgerSeq++
	s.Entry++
	if err = s.Repo.ApplyChange(Change{Asset: asset, Delta: delta, Balance: balance, Ledger: next.LedgerSeq, Entry: s.Entry, Reason: reason, Kind: kind}); err != nil {
		return err
	}
	*w = next
	return nil
}
func (s Service) Grant(p domain.Prize, round int, source string, sourceID uint64, no int, draw uint64) (uint64, error) {
	id, err := s.NewID()
	if err != nil {
		return 0, err
	}
	return id, s.Repo.SaveGrants([]Grant{{ID: id, Round: round, Prize: p, Source: source, SourceID: sourceID, No: no, DrawID: draw}})
}
func (s Service) Select(id, item string) (domain.Prize, string, uint64, error) {
	g, err := s.Repo.LockReward(id)
	if err != nil {
		return domain.Prize{}, "", 0, err
	}
	p, changed, err := g.Reward.Select(item)
	if err != nil || !changed {
		return p, "", 0, err
	}
	if err = s.Repo.MarkRewardSelected(id, p); err != nil {
		return p, "", 0, err
	}
	source, grant, err := s.Repo.RewardSource(id)
	return p, source, grant, err
}

func (s Service) InitializeAccount(event uint64) error {
	return s.Repo.InitializeAccount(event, domain.InitialWallet())
}
