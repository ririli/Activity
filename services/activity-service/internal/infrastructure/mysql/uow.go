package mysql

import (
	"activity/pkg/hourly"
	app "activity/services/activity-service/internal/application"
	assets "activity/services/activity-service/internal/assets/infrastructure/mysql"
	"activity/services/activity-service/internal/fault"
	"activity/services/activity-service/internal/infrastructure/sqlkit"
	lottery "activity/services/activity-service/internal/lottery/infrastructure/mysql"
	participation "activity/services/activity-service/internal/participation/infrastructure/mysql"
	"context"
	"database/sql"
	"fmt"
)

type UnitOfWork struct {
	DB    *sql.DB
	NewID func(context.Context) (uint64, error)
}

func (m UnitOfWork) session(ctx context.Context, tx *sql.Tx, s app.Scope, o *outbox) app.Session {
	port := sqlkit.Wrap(tx)
	a := &assets.Repository{Tx: port, Ctx: ctx, User: s.User, Edition: s.Edition, Biz: s.Biz, NewID: m.NewID}
	l := &lottery.Repository{Tx: port, Ctx: ctx, User: s.User, Edition: s.Edition, Biz: s.Biz, NewID: m.NewID}
	p := &participation.Repository{Tx: port, Ctx: ctx, User: s.User, Edition: s.Edition, Biz: s.Biz}
	return app.Session{Assets: a, Lottery: l, Participation: p, Outbox: o, AdmitHourly: o.admit(s), HourlyGrant: hourlyGrantStore{tx}, BindRound: func(round int, progress int64) {
		a.Round = round
		l.Round = round
		p.Round = round
		l.Progress = progress
	}}
}
func (m UnitOfWork) Within(ctx context.Context, s app.Scope, fn func(app.Session) error) error {
	// Mutations serialize on wallet/progress row locks. A consistent read made
	// before waiting for those locks must not pin an older MySQL snapshot.
	tx, err := m.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	o := &outbox{ctx: ctx, tx: tx}
	session := m.session(ctx, tx, s, o)
	if err = fn(session); err != nil {
		return err
	}
	if err = o.flush(s); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("%w: %v", fault.ErrCommitUnknown, err)
	}
	return nil
}
func (m UnitOfWork) Read(ctx context.Context, s app.Scope, fn func(app.Session) error) error {
	tx, err := m.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return fn(m.session(ctx, tx, s, &outbox{ctx: ctx, tx: tx}))
}

type outbox struct {
	ctx      context.Context
	tx       *sql.Tx
	position hourly.Position
	pending  []outboxEntry
}

type outboxEntry struct {
	event           app.Event
	user, aggregate uint64
}

func (o *outbox) Append(ev app.Event, user, aggregate uint64) error {
	if o.position.Cutoff != 0 && (ev.Type == "FormalDrawCommitted.v1" || ev.Type == "ParticipationInitialized.v1") {
		o.pending = append(o.pending, outboxEntry{ev, user, aggregate})
		return nil
	}
	_, err := o.tx.ExecContext(o.ctx, `INSERT INTO outbox_event(event_id,event_type,aggregate_id,user_id,payload) VALUES(?,?,?,?,?)`, ev.ID, ev.Type, aggregate, user, ev.Payload)
	return err
}

var _ app.UnitOfWork = UnitOfWork{}
