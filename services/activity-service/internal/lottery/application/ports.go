package application

import (
	domain "activity/services/activity-service/internal/lottery/domain"
	"time"
)

type DrawCommand struct {
	Edition, Operation, Pool string
	Count                    int
}
type DrawStep struct {
	domain.Step
	GrantID string `json:"grant_id"`
}
type DrawResult struct {
	DrawID    string     `json:"draw_id"`
	Results   []DrawStep `json:"results"`
	Consumed  int64      `json:"consumed_petals"`
	Refunded  int64      `json:"refunded_petals"`
	Requested int        `json:"requested_count"`
	Actual    int        `json:"actual_count"`
	Flowers   int        `json:"flowers"`
}
type Recovery struct {
	Hash   []byte
	Until  time.Time
	Result DrawResult
}
type DrawWrite struct {
	ID                 uint64
	Round, Version     int
	Mode               string
	PreviewID          uint64
	Pool, Operation    string
	Requested          int
	UnitCost, Charged  int64
	Plan               domain.Plan
	Seq                int64
	Occurred, Accepted time.Time
	RequestHash        [32]byte
	Idempotent         bool
}

// Repository is bound to one transaction and authenticated user/edition scope.
type Repository interface {
	FindPreview() (id uint64, raw []byte, status string, cost int64, err error)
	LockPreview() (domain.Preview, error)
	InsertPreview(id uint64, round, version, actual int, cost int64, snapshot, raw, expected, hash []byte) error
	ConfirmPreview(id, drawID uint64) error
	AbandonPreview() (int64, error)
	PreviewStatus() (string, error)
	SaveDraw(DrawWrite) error
	FindDraw(string) (Recovery, error)
	LoadDraw(uint64) (DrawResult, error)
}
