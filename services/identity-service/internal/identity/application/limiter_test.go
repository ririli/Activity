package application

import (
	"activity/pkg/ratelimit"
	"activity/services/identity-service/internal/identity/domain"
	"activity/services/identity-service/internal/identity/infrastructure/security"
	"context"
	"testing"
	"time"
)

type counterLimiter struct{ failures, registrations int }

func (l *counterLimiter) Register(context.Context, string) error {
	l.registrations++
	return &ratelimit.Rejection{Status: 429, Code: "RATE_LIMITED", After: time.Minute}
}
func (l *counterLimiter) LoginFailures(context.Context, string) (int, time.Duration, error) {
	return l.failures, 15 * time.Minute, nil
}
func (l *counterLimiter) RecordLoginFailure(context.Context, string) error { l.failures++; return nil }
func (l *counterLimiter) ClearLoginFailures(context.Context, string) error {
	l.failures = 0
	return nil
}

func TestLoginFifthFailureAndSubsequentLimit(t *testing.T) {
	hash, err := security.HashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	l := &counterLimiter{}
	s := Service{Repo: &fakeRepo{account: domain.Account{UserID: 42, PasswordHash: hash}}, Limiter: l, HashSlots: make(chan struct{}, 4)}
	for i := 0; i < 5; i++ {
		_, err := s.Login(context.Background(), Credentials{Account: "example", Password: "wrong-password"})
		assertFault(t, err, 401, "INVALID_CREDENTIALS")
	}
	_, err = s.Login(context.Background(), Credentials{Account: "example", Password: "wrong-password"})
	rejection, ok := err.(*ratelimit.Rejection)
	if !ok || rejection.Status != 429 || rejection.Code != "LOGIN_LIMITED" || rejection.After != 15*time.Minute {
		t.Fatalf("%v", err)
	}
	if l.failures != 5 {
		t.Fatalf("rejected attempt counted: %d", l.failures)
	}
}

func TestRegistrationValidatesBeforeCountingAndHashing(t *testing.T) {
	l := &counterLimiter{}
	s := Service{Limiter: l} // No repository: denied requests cannot reach persistence.
	_, err := s.Register(context.Background(), Credentials{}, "192.0.2.1")
	assertFault(t, err, 400, "INVALID_ACCOUNT")
	if l.registrations != 0 {
		t.Fatal("invalid request counted")
	}
	_, err = s.Register(context.Background(), Credentials{Account: "example", Password: "password-123"}, "192.0.2.1")
	if _, ok := err.(*ratelimit.Rejection); !ok {
		t.Fatalf("%v", err)
	}
	if l.registrations != 1 {
		t.Fatal("valid attempt not counted")
	}
}

func TestHashCapacityRejectsWithoutWaiting(t *testing.T) {
	s := Service{HashSlots: make(chan struct{}, 1)}
	release, err := s.hashSlot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	start := time.Now()
	_, err = s.hashSlot(context.Background())
	assertFault(t, err, 503, "PASSWORD_CAPACITY")
	if time.Since(start) > 100*time.Millisecond {
		t.Fatal("waited for occupied hash slot")
	}
}

func TestLoginRejectsAccountAliasesBeforeDependencies(t *testing.T) {
	s := Service{} // Invalid names must not reach Redis, hashing or MySQL.
	for _, name := range []string{"example ", " example", "example\t", "例子", "ab", ""} {
		_, err := s.Login(context.Background(), Credentials{Account: name, Password: "password-123"})
		assertFault(t, err, 400, "INVALID_REQUEST")
	}
}
