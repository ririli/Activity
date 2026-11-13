package application

import (
	"activity/services/activity-service/internal/fault"
	domain "activity/services/activity-service/internal/lottery/domain"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"time"
)

type Service struct {
	Repo   Repository
	Random func(int64) (int64, error)
	Now    func() time.Time
}

func (s Service) Generate(p domain.Pool, flowers int, value int64, count int) (domain.Plan, error) {
	return domain.Simulate(p, domain.Progress{Flowers: flowers, GoldValue: value}, count, s.Random)
}
func RequestHash(c DrawCommand) [32]byte {
	raw, _ := json.Marshal([]any{c.Edition, c.Pool, c.Count, "NORMAL"})
	return sha256.Sum256(raw)
}
func CanonicalHash(raw []byte) ([32]byte, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return [32]byte{}, err
	}
	canonical, err := json.Marshal(v)
	return sha256.Sum256(canonical), err
}
func (s Service) Recover(c DrawCommand) (DrawResult, bool, error) {
	r, err := s.Repo.FindDraw(c.Operation)
	if errors.Is(err, fault.ErrNotFound) {
		return DrawResult{}, false, nil
	}
	if err != nil {
		return DrawResult{}, false, err
	}
	hash := RequestHash(c)
	if string(r.Hash) != string(hash[:]) {
		return DrawResult{}, false, fault.New("IDEMPOTENCY_CONFLICT")
	}
	if !r.Until.After(s.Now()) {
		return DrawResult{}, false, fault.New("IDEMPOTENCY_EXPIRED")
	}
	return r.Result, true, nil
}
