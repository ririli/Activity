package http

import (
	"activity/pkg/ratelimit"
	app "activity/services/activity-service/internal/application"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type recordingLimiter struct {
	keys   []string
	result ratelimit.Result
	err    error
}

func (l *recordingLimiter) Allow(_ context.Context, key string, _ ratelimit.Rule) (ratelimit.Result, error) {
	l.keys = append(l.keys, key)
	return l.result, l.err
}

type protectedUseCases struct {
	UseCases
	calls int
}

func (s *protectedUseCases) Draw(context.Context, app.DrawCommand) (any, error) {
	s.calls++
	return "draw", nil
}
func (s *protectedUseCases) Join(context.Context, app.JoinCommand) (any, error) {
	s.calls++
	return "join", nil
}
func (s *protectedUseCases) Read(context.Context, string, string, string, string, string) (any, error) {
	s.calls++
	return "read", nil
}

func TestRouteAdmissionBeforeUseCase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "access.pub")
	if err := os.WriteFile(path, []byte(hex.EncodeToString(public)), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ACCESS_PUBLIC_KEY_FILE", path)
	t.Setenv("APP_ENV", "limit-test")
	identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"user_id":"42","session_id":"session","nickname":"name"}`))
	}))
	defer identity.Close()
	t.Setenv("IDENTITY_URL", identity.URL)
	now := time.Now().Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{"iss": "activity-identity", "aud": "activity-api", "sub": "42", "sid": "session", "jti": "id", "nbf": now - 1, "iat": now - 1, "exp": now + 60, "token_use": "access"})
	token.Header["kid"] = "v1"
	raw, err := token.SignedString(private)
	if err != nil {
		t.Fatal(err)
	}
	s := &protectedUseCases{}
	l := &recordingLimiter{result: ratelimit.Result{Allowed: false, RetryAfter: 1200 * time.Millisecond}}
	g := gin.New()
	Routes(g, s, l)
	for _, tc := range []struct{ method, path, group string }{{"POST", "/draw", "operation"}, {"POST", "/join", "join"}, {"GET", "/records/state", "query"}} {
		req := httptest.NewRequest(tc.method, "/api/activity/v1"+tc.path, strings.NewReader(`{"edition_id":"8","operation_id":"same","count":30,"user_id":"attacker"}`))
		req.Header.Set("Authorization", "Bearer "+raw)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		g.ServeHTTP(w, req)
		if w.Code != 429 || w.Header().Get("Retry-After") != "2" {
			t.Fatalf("%s: %d %s", tc.path, w.Code, w.Body.String())
		}
		if key := l.keys[len(l.keys)-1]; key != "rl:limit-test:activity:"+tc.group+":42" {
			t.Fatalf("untrusted subject or incorrect grouping: %s", key)
		}
	}
	if s.calls != 0 {
		t.Fatalf("denied requests invoked business: %d", s.calls)
	}
	l.err = errors.New("redis offline")
	for _, tc := range []struct {
		method, path string
		status       int
	}{{"GET", "/records/state", 200}, {"POST", "/join", 503}, {"POST", "/draw", 503}} {
		req := httptest.NewRequest(tc.method, "/api/activity/v1"+tc.path, strings.NewReader(`{"edition_id":"8"}`))
		req.Header.Set("Authorization", "Bearer "+raw)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		g.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Fatalf("%s: %d %s", tc.path, w.Code, w.Body.String())
		}
	}
	if s.calls != 1 {
		t.Fatalf("only query should run during failure: %d", s.calls)
	}
}
