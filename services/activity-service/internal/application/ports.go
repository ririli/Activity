package application

import (
	assets "activity/services/activity-service/internal/assets/application"
	lottery "activity/services/activity-service/internal/lottery/application"
	participation "activity/services/activity-service/internal/participation/application"
	"context"
	"encoding/json"
	"time"
)

// Scope is immutable transaction metadata, never a shared business state.
type Scope struct{ User, Edition, Biz uint64 }
type Event struct {
	ID      string          `json:"event_id"`
	Type    string          `json:"event_type"`
	Payload json.RawMessage `json:"payload"`
}
type Outbox interface {
	Append(Event, uint64, uint64) error
}

// Session exposes capabilities backed by one connection and transaction.
type Session struct {
	Assets        assets.Repository
	Lottery       lottery.Repository
	Participation participation.Repository
	Outbox        Outbox
	BindRound     func(int, int64)
	AdmitHourly   func(time.Time) (time.Time, error)
	HourlyGrant   HourlyGrantStore
}
type UnitOfWork interface {
	Within(context.Context, Scope, func(Session) error) error
	Read(context.Context, Scope, func(Session) error) error
}
type UserLock interface {
	Acquire(context.Context, string) (func(), error)
}
type Effects interface {
	Enqueue([]map[string]any)
}
type Queries interface {
	Read(context.Context, string, string, string, string, string) (any, error)
	Internal(context.Context, string, string, string, string) (any, error)
	BootstrapCandidates(context.Context) ([]JoinCommand, error)
}
type Runtime struct {
	NewID  func() uint64
	Now    func() time.Time
	Random func(int64) (int64, error)
}
