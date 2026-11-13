package application

import (
	"activity/pkg/platform"
	"activity/services/identity-service/internal/identity/domain"
	"activity/services/identity-service/internal/identity/infrastructure/security"
	"context"
	"crypto/ed25519"
	"errors"
	"testing"
	"time"
)

type fakeRepo struct {
	session      domain.Session
	account      domain.Account
	sessionErr   error
	rotate       Rotation
	rotateErr    error
	revokeExpiry time.Time
	revokeErr    error
	sessionCalls int
	revokeCalls  int
}

func (f *fakeRepo) Register(context.Context, domain.Account, domain.Session, uint64, []byte) error {
	return nil
}
func (f *fakeRepo) AccountByName(context.Context, string) (domain.Account, error) {
	return f.account, nil
}
func (f *fakeRepo) SessionAccount(context.Context, string) (domain.Session, domain.Account, error) {
	f.sessionCalls++
	return f.session, f.account, f.sessionErr
}
func (f *fakeRepo) CreateSession(context.Context, domain.Session) error { return nil }
func (f *fakeRepo) Rotate(context.Context, string, [32]byte, [32]byte) (Rotation, error) {
	return f.rotate, f.rotateErr
}
func (f *fakeRepo) Revoke(context.Context, string) (time.Time, error) {
	f.revokeCalls++
	return f.revokeExpiry, f.revokeErr
}
func (f *fakeRepo) PublicProfiles(context.Context, []string) (map[string]string, error) {
	return nil, nil
}

type fakeRevocation struct {
	revoked    bool
	err        error
	calls      int
	marked     int
	forgot     int
	lastExpiry time.Time
}

func (f *fakeRevocation) Revoked(context.Context, string, time.Time) (bool, error) {
	f.calls++
	return f.revoked, f.err
}
func (f *fakeRevocation) MarkRevoked(_ context.Context, _ string, expires time.Time) error {
	f.marked++
	f.lastExpiry = expires
	return nil
}
func (f *fakeRevocation) Forget(string) { f.forgot++ }

func testSigner(kind, audience string) security.TokenSigner {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	key := ed25519.NewKeyFromSeed(seed)
	return security.TokenSigner{
		Issuer:     "activity-identity",
		Audience:   audience,
		TokenUse:   kind,
		KeyID:      "v1",
		PrivateKey: key,
		PublicKeys: map[string]ed25519.PublicKey{"v1": key.Public().(ed25519.PublicKey)},
	}
}

func testService(repo *fakeRepo, rev *fakeRevocation) (*Service, string, string) {
	access := testSigner("access", "activity-api")
	refresh := testSigner("refresh", "identity-refresh")
	s := &Service{Repo: repo, Access: access, Refresh: refresh, Revocations: rev}
	now := time.Now().UTC()
	at, err := access.Issue("42", "sid", now, now.Add(15*time.Minute))
	if err != nil {
		panic(err)
	}
	rt, err := refresh.Issue("42", "sid", now, now.Add(time.Hour))
	if err != nil {
		panic(err)
	}
	return s, at, rt
}

func activeFixture() (*fakeRepo, *fakeRevocation) {
	expires := time.Now().UTC().Add(time.Hour)
	return &fakeRepo{
		session: domain.Session{ID: "sid", UserID: 42, ExpiresAt: expires},
		account: domain.Account{UserID: 42, Nickname: "nick", Status: "ACTIVE"},
	}, &fakeRevocation{}
}

func assertFault(t *testing.T, err error, status int, code string) {
	t.Helper()
	var f *platform.Fault
	if !errors.As(err, &f) {
		t.Fatalf("expected fault, got %v", err)
	}
	if f.Status != status || f.Code != code {
		t.Fatalf("fault = %d/%s, want %d/%s", f.Status, f.Code, status, code)
	}
}

func TestIntrospectChecksRevocationAndMySQL(t *testing.T) {
	repo, rev := activeFixture()
	s, token, _ := testService(repo, rev)
	p, err := s.Introspect(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if p.UserID != "42" || p.SessionID != "sid" || p.Nickname != "nick" {
		t.Fatalf("unexpected principal %+v", p)
	}
	if rev.calls != 1 || repo.sessionCalls != 1 {
		t.Fatalf("calls rev=%d session=%d", rev.calls, repo.sessionCalls)
	}
}

func TestIntrospectRevokedSkipsMySQL(t *testing.T) {
	repo, rev := activeFixture()
	rev.revoked = true
	s, token, _ := testService(repo, rev)
	_, err := s.Introspect(context.Background(), token)
	assertFault(t, err, 401, "INVALID_SESSION")
	if repo.sessionCalls != 0 {
		t.Fatal("revoked shortcut must not query mysql")
	}
}

func TestIntrospectRedisErrorFallsBackToMySQL(t *testing.T) {
	repo, rev := activeFixture()
	rev.err = errors.New("redis down")
	s, token, _ := testService(repo, rev)
	if _, err := s.Introspect(context.Background(), token); err != nil {
		t.Fatalf("expected mysql fallback, got %v", err)
	}
	if repo.sessionCalls != 1 {
		t.Fatal("mysql session check was skipped")
	}
}

func TestIntrospectMySQLRevocationBackfillsRedis(t *testing.T) {
	repo, rev := activeFixture()
	now := time.Now().UTC()
	repo.session.RevokedAt = &now
	repo.session.ExpiresAt = now.Add(time.Hour)
	s, token, _ := testService(repo, rev)
	_, err := s.Introspect(context.Background(), token)
	assertFault(t, err, 401, "INVALID_SESSION")
	if rev.marked != 1 {
		t.Fatalf("expected revocation backfill, got %d", rev.marked)
	}
}

func TestLogoutNoCookieIsIdempotent(t *testing.T) {
	repo, rev := activeFixture()
	s, _, _ := testService(repo, rev)
	if err := s.Logout(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	if repo.revokeCalls != 0 || rev.marked != 0 {
		t.Fatal("empty logout must not touch session or cache")
	}
}

func TestLogoutMarksRevocationAfterMySQL(t *testing.T) {
	repo, rev := activeFixture()
	expires := time.Now().UTC().Add(time.Hour)
	repo.revokeExpiry = expires
	s, _, refresh := testService(repo, rev)
	if err := s.Logout(context.Background(), refresh); err != nil {
		t.Fatal(err)
	}
	if repo.revokeCalls != 1 || rev.marked != 1 {
		t.Fatalf("calls revoke=%d mark=%d", repo.revokeCalls, rev.marked)
	}
	if !rev.lastExpiry.Equal(expires) {
		t.Fatalf("expiry = %v, want %v", rev.lastExpiry, expires)
	}
}

func TestLogoutMySQLFailureDoesNotClaimSuccess(t *testing.T) {
	repo, rev := activeFixture()
	repo.revokeErr = errors.New("db down")
	s, _, refresh := testService(repo, rev)
	if err := s.Logout(context.Background(), refresh); err == nil {
		t.Fatal("expected error")
	}
	if rev.marked != 0 {
		t.Fatal("must not write revocation cache when mysql failed")
	}
}

func TestRenewReplayMarksRevocation(t *testing.T) {
	repo, rev := activeFixture()
	expires := time.Now().UTC().Add(time.Hour)
	repo.rotate = Rotation{ExpiresAt: expires, Replayed: true}
	repo.rotateErr = platform.Fail(401, "REFRESH_REPLAY")
	s, _, refresh := testService(repo, rev)
	_, err := s.Renew(context.Background(), refresh)
	assertFault(t, err, 401, "REFRESH_REPLAY")
	if rev.marked != 1 {
		t.Fatalf("expected replay revocation, got %d", rev.marked)
	}
}

func TestRenewSuccessDoesNotMarkRevocation(t *testing.T) {
	repo, rev := activeFixture()
	repo.rotate = Rotation{ExpiresAt: time.Now().UTC().Add(time.Hour)}
	s, _, refresh := testService(repo, rev)
	if _, err := s.Renew(context.Background(), refresh); err != nil {
		t.Fatal(err)
	}
	if rev.marked != 0 {
		t.Fatal("successful rotation must not revoke")
	}
}

func TestRenewRejectsInvalidSessionBeforeIssuingTokens(t *testing.T) {
	for _, state := range []string{"expired", "revoked", "disabled"} {
		t.Run(state, func(t *testing.T) {
			repo, rev := activeFixture()
			s, _, refresh := testService(repo, rev)
			now := time.Now()
			switch state {
			case "expired":
				repo.session.ExpiresAt = now.Add(-time.Second)
			case "revoked":
				repo.session.RevokedAt = &now
			case "disabled":
				repo.account.Status = "DISABLED"
			}
			_, err := s.Renew(context.Background(), refresh)
			assertFault(t, err, 401, "INVALID_SESSION")
		})
	}
}
