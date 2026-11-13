package mysql

import (
	app "activity/services/activity-service/internal/application"
	assets "activity/services/activity-service/internal/assets/application"
	ad "activity/services/activity-service/internal/assets/domain"
	astore "activity/services/activity-service/internal/assets/infrastructure/mysql"
	"activity/services/activity-service/internal/fault"
	lottery "activity/services/activity-service/internal/lottery/application"
	ld "activity/services/activity-service/internal/lottery/domain"
	lstore "activity/services/activity-service/internal/lottery/infrastructure/mysql"
	pd "activity/services/activity-service/internal/participation/domain"
	pstore "activity/services/activity-service/internal/participation/infrastructure/mysql"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

type statement struct {
	query string
	args  []driver.NamedValue
}
type recordingDB struct {
	pending, committed                        []statement
	begins, commits, rollbacks, execs, failAt int
	active, unknown                           bool
}
type connector struct{ state *recordingDB }

func (c connector) Connect(context.Context) (driver.Conn, error) { return &recordingConn{c.state}, nil }
func (c connector) Driver() driver.Driver                        { return recordingDriver{c.state} }

type recordingDriver struct{ state *recordingDB }

func (d recordingDriver) Open(string) (driver.Conn, error) { return &recordingConn{d.state}, nil }

type recordingConn struct{ s *recordingDB }

func (c *recordingConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (c *recordingConn) Close() error { return nil }
func (c *recordingConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
func (c *recordingConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	if c.s.active {
		return nil, errors.New("nested transaction")
	}
	c.s.active = true
	c.s.begins++
	return recordingTx{c.s}, nil
}
func (c *recordingConn) ExecContext(_ context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	if !c.s.active {
		return nil, errors.New("write outside transaction")
	}
	if strings.Count(q, "?") != len(args) {
		return nil, fmt.Errorf("placeholder mismatch %d/%d: %s", strings.Count(q, "?"), len(args), q)
	}
	c.s.execs++
	if c.s.failAt == c.s.execs {
		return nil, errors.New("injected SQL failure")
	}
	c.s.pending = append(c.s.pending, statement{q, append([]driver.NamedValue(nil), args...)})
	return driver.RowsAffected(1), nil
}
func (c *recordingConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return emptyRows{}, nil
}

type emptyRows struct{}

func (emptyRows) Columns() []string         { return []string{"id"} }
func (emptyRows) Close() error              { return nil }
func (emptyRows) Next([]driver.Value) error { return io.EOF }

type recordingTx struct{ s *recordingDB }

func (tx recordingTx) Commit() error {
	tx.s.active = false
	tx.s.commits++
	tx.s.committed = append(tx.s.committed, tx.s.pending...)
	tx.s.pending = nil
	if tx.s.unknown {
		return errors.New("connection lost during commit")
	}
	return nil
}
func (tx recordingTx) Rollback() error {
	tx.s.active = false
	tx.s.rollbacks++
	tx.s.pending = nil
	return nil
}
func TestTransactionFactorySharesConnectionAndRollsBackEveryWrite(t *testing.T) {
	run := func(t *testing.T, failAt int) (*recordingDB, error) {
		state := &recordingDB{failAt: failAt}
		db := sql.OpenDB(connector{state})
		defer db.Close()
		now := time.Now()
		m := UnitOfWork{DB: db}
		err := m.Within(context.Background(), app.Scope{User: 7, Edition: 8, Biz: 9}, func(s app.Session) error {
			a := s.Assets.(*astore.Repository)
			l := s.Lottery.(*lstore.Repository)
			p := s.Participation.(*pstore.Repository)
			if a.Tx != l.Tx || a.Tx != p.Tx {
				t.Fatal("module repositories use different transactions")
			}
			s.BindRound(2, 3)
			if a.Round != 2 || l.Round != 2 || p.Round != 2 || l.Progress != 3 {
				t.Fatal("round scope not bound")
			}
			if err := s.Assets.ApplyChange(assets.Change{Asset: ad.Petal, Delta: -900, Balance: 100, Ledger: 1, Entry: 1, Reason: "DRAW_CHARGE", Kind: "DRAW"}); err != nil {
				return err
			}
			grants := []assets.Grant{}
			steps := []ld.Step{}
			for i := 0; i < 180; i++ {
				grants = append(grants, assets.Grant{ID: uint64(100 + i), Round: 2, Prize: ad.Prize{ID: 1001, Code: "1001", Name: "gold", Quantity: 1}, Source: "DAY_DRAW", SourceID: 9, No: i + 1, DrawID: 9})
				steps = append(steps, ld.Step{Prize: ld.Prize{ID: 1001, Code: "1001", Name: "gold", Quantity: 1}, Before: i / 10, After: (i + 1) / 10, GoldValue: int64(i+1) * 300})
			}
			if err := s.Assets.SaveGrants(grants); err != nil {
				return err
			}
			if err := s.Participation.SaveDrawProgress(pd.Progress{Round: 2, Flowers: 18, Value: 54000, Unlocked: 3, Score: 900, Seq: 1, Reached: now.Unix()}); err != nil {
				return err
			}
			if err := s.Lottery.SaveDraw(lottery.DrawWrite{ID: 9, Round: 2, Version: 1, Mode: "NORMAL", Pool: "DAY", Operation: "test", Requested: 180, UnitCost: 5, Charged: 900, Plan: ld.Plan{Steps: steps, Cost: 900, Flowers: 18, GoldValue: 54000}, Seq: 1, Occurred: now, Accepted: now, Idempotent: true}); err != nil {
				return err
			}
			return s.Outbox.Append(app.Event{ID: "10", Type: "FormalDrawCommitted.v1", Payload: []byte(`{}`)}, 7, 9)
		})
		return state, err
	}
	state, err := run(t, 0)
	if err != nil {
		t.Fatal(err)
	}
	if state.begins != 1 || state.commits != 1 || state.rollbacks != 0 || len(state.committed) == 0 {
		t.Fatalf("transaction %+v", state)
	}
	details, rewards, flowers := 0, 0, 0
	for _, st := range state.committed {
		switch {
		case strings.HasPrefix(st.query, "INSERT INTO draw_detail_3"):
			details += len(st.args) / 14
		case strings.HasPrefix(st.query, "INSERT INTO reward_grant"):
			rewards += len(st.args) / 16
		case strings.HasPrefix(st.query, "INSERT INTO flower_lit_log_3"):
			flowers += len(st.args) / 9
		}
	}
	if details != 180 || rewards != 180 || flowers != 18 {
		t.Fatalf("lost facts: details=%d rewards=%d lights=%d", details, rewards, flowers)
	}
	for i := 1; i <= state.execs; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			failed, err := run(t, i)
			if err == nil || failed.commits != 0 || failed.rollbacks != 1 || len(failed.committed) != 0 || len(failed.pending) != 0 {
				t.Fatalf("partial SQL commit at %d: %+v %v", i, failed, err)
			}
		})
	}
}
func TestCommitUnknownAndReadOnlyRollback(t *testing.T) {
	state := &recordingDB{unknown: true}
	db := sql.OpenDB(connector{state})
	defer db.Close()
	u := UnitOfWork{DB: db}
	if err := u.Within(context.Background(), app.Scope{}, func(app.Session) error { return nil }); !errors.Is(err, fault.ErrCommitUnknown) {
		t.Fatalf("unknown outcome lost %v", err)
	}
	if err := u.Read(context.Background(), app.Scope{User: 1}, func(s app.Session) error {
		_, err := s.Assets.FindRankingGrant(2)
		if !errors.Is(err, fault.ErrNotFound) {
			t.Fatalf("adapter leaked SQL missing-row error: %v", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if state.commits != 1 || state.rollbacks != 1 {
		t.Fatalf("read transaction not released %+v", state)
	}
}
