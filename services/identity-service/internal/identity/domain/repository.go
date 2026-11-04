package domain

import (
	"context"
	"time"
)

type Account struct {
	UserID       uint64
	Name         string
	Nickname     string
	PasswordHash string
	Status       string
}

// PublicProfile is the only account projection exposed to other services.
type PublicProfile struct {
	UserID      string
	DisplayName string
}

type Session struct {
	ID          string
	UserID      uint64
	RefreshHash [32]byte
	ExpiresAt   time.Time
	RevokedAt   *time.Time
}

// Repository owns account and session state; implementations must commit the
// account, initial session and account-created event in one transaction.
type Repository interface {
	Register(context.Context, Account, Session, uint64, []byte) error
	AccountByName(context.Context, string) (Account, error)
	SessionAccount(context.Context, string) (Session, Account, error)
}
