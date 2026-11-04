package uow

import (
	"context"
	"database/sql"
	"errors"
)

type Result interface {
	RowsAffected() (int64, error)
}

type Row interface {
	Scan(dest ...any) error
}

type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close() error
}

// Tx is the transaction-scoped SQL port used by application persistence
// helpers. It deliberately exposes no Begin/Commit/Rollback methods.
type Tx interface {
	Exec(ctx context.Context, query string, args ...any) (Result, error)
	Query(ctx context.Context, query string, args ...any) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) Row
}

// UnitOfWork owns transaction boundaries. Application services call Within for
// mutating use cases and Read for read-only snapshots.
type UnitOfWork interface {
	Within(ctx context.Context, fn func(Tx) error) error
	Read(ctx context.Context, fn func(Tx) error) error
}

var ErrCommitUnknown = errors.New("transaction commit outcome unknown")

// ErrNoRows is re-exported so application/state packages do not need to import
// database/sql directly.
var ErrNoRows = sql.ErrNoRows
