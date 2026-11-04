package application

import (
	"activity/pkg/platform"
	"activity/services/identity-service/internal/identity/domain"
	"activity/services/identity-service/internal/identity/infrastructure/security"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"strconv"
	"time"
)

type Repository interface {
	domain.Repository
	CreateSession(context.Context, domain.Session) error
	Rotate(context.Context, string, [32]byte, [32]byte) error
	Revoke(context.Context, string) error
	Profiles(context.Context, []string) (map[string]string, error)
}
type Limiter interface {
	Take(context.Context, string, int, time.Duration) error
	Failures(context.Context, string) (int, error)
	Clear(context.Context, string) error
}
type Service struct {
	Repo            Repository
	Access, Refresh security.TokenSigner
	Limiter         Limiter
	HashSlots       chan struct{}
	DummyHash       string
}
type Credentials struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname"`
}
type Tokens struct {
	Access   string `json:"access_token"`
	Refresh  string `json:"-"`
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
}

func (s *Service) hashSlot(ctx context.Context) (func(), error) {
	select {
	case s.HashSlots <- struct{}{}:
		return func() { <-s.HashSlots }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (s *Service) issue(a domain.Account, sid string, expires time.Time) (Tokens, error) {
	now := time.Now().UTC()
	accessExp := now.Add(15 * time.Minute)
	if expires.Before(accessExp) {
		accessExp = expires
	}
	aToken, e := s.Access.Issue(strconv.FormatUint(a.UserID, 10), sid, now, accessExp)
	if e != nil {
		return Tokens{}, e
	}
	rToken, e := s.Refresh.Issue(strconv.FormatUint(a.UserID, 10), sid, now, expires)
	return Tokens{aToken, rToken, strconv.FormatUint(a.UserID, 10), a.Nickname}, e
}
func (s *Service) Register(ctx context.Context, c Credentials, ip string) (Tokens, error) {
	if e := domain.ValidateAccount(c.Account, c.Password, c.Nickname); e != nil {
		return Tokens{}, platform.Fail(400, "INVALID_ACCOUNT")
	}
	if e := s.Limiter.Take(ctx, "register:"+ip, 3, time.Minute); e != nil {
		return Tokens{}, e
	}
	release, e := s.hashSlot(ctx)
	if e != nil {
		return Tokens{}, e
	}
	hash, e := security.HashPassword(c.Password)
	release()
	if e != nil {
		return Tokens{}, e
	}
	nick := c.Nickname
	if nick == "" {
		nick = c.Account
	}
	a := domain.Account{UserID: platform.ID(), Name: c.Account, Nickname: nick, PasswordHash: hash, Status: "ACTIVE"}
	sid := platform.Random()
	expiry := time.Now().UTC().Add(7 * 24 * time.Hour)
	tokens, e := s.issue(a, sid, expiry)
	if e != nil {
		return Tokens{}, e
	}
	session := domain.Session{ID: sid, UserID: a.UserID, RefreshHash: sha256.Sum256([]byte(tokens.Refresh)), ExpiresAt: expiry}
	e = s.Repo.Register(ctx, a, session, platform.ID(), platform.JSON(map[string]string{"user_id": tokens.UserID}))
	if e != nil { // Only the account-name uniqueness constraint is an expected conflict.
		var lookup domain.Account
		lookup, check := s.Repo.AccountByName(ctx, c.Account)
		if check == nil && lookup.UserID != 0 {
			return Tokens{}, platform.Fail(409, "ACCOUNT_EXISTS")
		}
		return Tokens{}, e
	}
	return tokens, nil
}
func (s *Service) Login(ctx context.Context, c Credentials) (Tokens, error) {
	if len(c.Account) > 20 || len(c.Password) > 512 {
		return Tokens{}, platform.Fail(400, "INVALID_REQUEST")
	}
	n, e := s.Limiter.Failures(ctx, c.Account)
	if e != nil {
		return Tokens{}, e
	}
	if n >= 5 {
		return Tokens{}, platform.Fail(429, "LOGIN_LIMITED")
	}
	a, e := s.Repo.AccountByName(ctx, c.Account)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return Tokens{}, e
	}
	hash := a.PasswordHash
	if errors.Is(e, sql.ErrNoRows) {
		hash = s.DummyHash
	}
	release, e := s.hashSlot(ctx)
	if e != nil {
		return Tokens{}, e
	}
	ok, e := security.VerifyPassword(hash, c.Password)
	release()
	if e != nil {
		return Tokens{}, e
	}
	if !ok || a.UserID == 0 {
		if e = s.Limiter.Take(ctx, "login:"+c.Account, 1000000, 15*time.Minute); e != nil {
			return Tokens{}, e
		}
		return Tokens{}, platform.Fail(401, "INVALID_CREDENTIALS")
	}
	if a.Status != "ACTIVE" {
		return Tokens{}, platform.Fail(403, "ACCOUNT_DISABLED")
	}
	if e = s.Limiter.Clear(ctx, c.Account); e != nil {
		return Tokens{}, e
	}
	sid := platform.Random()
	expires := time.Now().UTC().Add(7 * 24 * time.Hour)
	t, e := s.issue(a, sid, expires)
	if e != nil {
		return Tokens{}, e
	}
	e = s.Repo.CreateSession(ctx, domain.Session{ID: sid, UserID: a.UserID, RefreshHash: sha256.Sum256([]byte(t.Refresh)), ExpiresAt: expires})
	return t, e
}
func (s *Service) Introspect(ctx context.Context, raw string) (platform.Principal, error) {
	c, e := s.Access.Verify(raw)
	if e != nil {
		return platform.Principal{}, platform.Fail(401, "INVALID_TOKEN")
	}
	session, a, e := s.Repo.SessionAccount(ctx, c.SessionID)
	if errors.Is(e, sql.ErrNoRows) {
		return platform.Principal{}, platform.Fail(401, "INVALID_SESSION")
	}
	if e != nil {
		return platform.Principal{}, e
	}
	if session.RevokedAt != nil || !session.ExpiresAt.After(time.Now()) || a.Status != "ACTIVE" || strconv.FormatUint(a.UserID, 10) != c.Subject {
		return platform.Principal{}, platform.Fail(401, "INVALID_SESSION")
	}
	return platform.Principal{UserID: c.Subject, SessionID: c.SessionID, Nickname: a.Nickname}, nil
}
func (s *Service) Renew(ctx context.Context, raw string) (Tokens, error) {
	c, e := s.Refresh.Verify(raw)
	if e != nil {
		return Tokens{}, platform.Fail(401, "INVALID_REFRESH")
	}
	session, a, e := s.Repo.SessionAccount(ctx, c.SessionID)
	if errors.Is(e, sql.ErrNoRows) {
		return Tokens{}, platform.Fail(401, "INVALID_SESSION")
	}
	if e != nil {
		return Tokens{}, e
	}
	if strconv.FormatUint(a.UserID, 10) != c.Subject {
		return Tokens{}, platform.Fail(401, "INVALID_REFRESH")
	}
	t, e := s.issue(a, session.ID, session.ExpiresAt)
	if e != nil {
		return Tokens{}, e
	}
	e = s.Repo.Rotate(ctx, session.ID, sha256.Sum256([]byte(raw)), sha256.Sum256([]byte(t.Refresh)))
	return t, e
}
func (s *Service) Logout(ctx context.Context, raw string) error {
	c, e := s.Refresh.Verify(raw)
	if e != nil {
		return platform.Fail(401, "INVALID_REFRESH")
	}
	return s.Repo.Revoke(ctx, c.SessionID)
}
