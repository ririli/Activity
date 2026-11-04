package uow

import (
	"context"
	"database/sql"
	"fmt"
)

type MySQL struct{ DB *sql.DB }

func (m MySQL) Within(ctx context.Context, fn func(Tx) error) error {
	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(sqlTx{tx: tx}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: %v", ErrCommitUnknown, err)
	}
	return nil
}

func (m MySQL) Read(ctx context.Context, fn func(Tx) error) error {
	tx, err := m.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return fn(sqlTx{tx: tx})
}

type sqlTx struct{ tx *sql.Tx }

func (s sqlTx) Exec(ctx context.Context, query string, args ...any) (Result, error) {
	return s.tx.ExecContext(ctx, query, args...)
}
func (s sqlTx) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	return s.tx.QueryContext(ctx, query, args...)
}
func (s sqlTx) QueryRow(ctx context.Context, query string, args ...any) Row {
	return s.tx.QueryRowContext(ctx, query, args...)
}

// Compile-time checks.
var _ UnitOfWork = MySQL{}
var _ Tx = sqlTx{}
