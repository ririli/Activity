package domain

import "activity/services/activity-service/internal/fault"

type Preview struct {
	ID                  uint64
	Round               int
	Version             int64
	Status              string
	Raw, Expected, Hash []byte
	DrawID              uint64
}

func (p Preview) CheckConfirm(round int, progress, formal int64) error {
	if p.Status == "CONFIRMED" {
		return nil
	}
	if p.Status != "PENDING" || formal != 0 || p.Round != round || p.Version != progress {
		return fault.New("PREVIEW_STALE")
	}
	return nil
}
