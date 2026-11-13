package sqlkit

import (
	"context"
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

// Tx is an infrastructure-only SQL adapter port. Business packages use their
// context repository interfaces and never receive this interface.
type Tx interface {
	Exec(ctx context.Context, query string, args ...any) (Result, error)
	Query(ctx context.Context, query string, args ...any) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) Row
}
