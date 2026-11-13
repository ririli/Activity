package sqlkit

import (
	"activity/services/activity-service/internal/fault"
	"context"
	"database/sql"
	"errors"
)

type DBTX interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type wrapper struct{ tx DBTX }

func Wrap(tx DBTX) Tx { return wrapper{tx} }
func (w wrapper) Exec(c context.Context, q string, a ...any) (Result, error) {
	return w.tx.ExecContext(c, q, a...)
}
func (w wrapper) Query(c context.Context, q string, a ...any) (Rows, error) {
	return w.tx.QueryContext(c, q, a...)
}

type row struct{ r *sql.Row }

func (r row) Scan(v ...any) error {
	err := r.r.Scan(v...)
	if errors.Is(err, sql.ErrNoRows) {
		return fault.ErrNotFound
	}
	return err
}
func (w wrapper) QueryRow(c context.Context, q string, a ...any) Row {
	return row{w.tx.QueryRowContext(c, q, a...)}
}
