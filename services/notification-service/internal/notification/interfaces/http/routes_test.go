package http

import (
	"activity/pkg/ratelimit"
	"activity/services/notification-service/internal/notification/application"
	"activity/services/notification-service/internal/notification/domain"
	"bufio"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type allowLimiter struct{}

func (allowLimiter) Allow(context.Context, string, ratelimit.Rule) (ratelimit.Result, error) {
	return ratelimit.Result{Allowed: true}, nil
}

func TestSSELifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Stable test-only key also works with platform's process-wide key cache
	// when this suite is executed with -count.
	private := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	path := filepath.Join(t.TempDir(), "access.pub")
	if err := os.WriteFile(path, []byte(hex.EncodeToString(private.Public().(ed25519.PublicKey))), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ACCESS_PUBLIC_KEY_FILE", path)
	t.Setenv("INTERNAL_TOKEN", strings.Repeat("test", 8))
	identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Token") != strings.Repeat("test", 8) {
			w.WriteHeader(401)
			return
		}
		_, _ = io.WriteString(w, `{"user_id":"1","session_id":"sid","nickname":"test"}`)
	}))
	defer identity.Close()
	t.Setenv("IDENTITY_URL", identity.URL)
	now := time.Now()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{"iss": "activity-identity", "aud": "activity-api", "exp": now.Add(time.Hour).Unix(), "iat": now.Unix(), "nbf": now.Unix(), "sid": "sid", "jti": "test", "token_use": "access"})
	jwtToken.Header["kid"] = "v1"
	token, err := jwtToken.SignedString(private)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"close", "disconnect", "unavailable", "session_limit", "capacity"} {
		t.Run(mode, func(t *testing.T) {
			cfg := application.DefaultConfig()
			if mode == "capacity" {
				cfg.MaxConnections = 1
			}
			h, err := application.New(context.Background(), cfg, EncodeFrame)
			if err != nil {
				t.Fatal(err)
			}
			defer h.Close()
			h.SetReady(true)
			g := gin.New()
			Routes(g, h, allowLimiter{})
			server := httptest.NewServer(g)
			defer server.Close()
			if mode == "session_limit" || mode == "capacity" {
				n := 5
				if mode == "capacity" {
					n = 1
				}
				for i := 0; i < n; i++ {
					if _, err = h.Subscribe("sid"); err != nil {
						t.Fatal(err)
					}
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/notification/v1/barrage/stream", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if mode == "session_limit" || mode == "capacity" {
				want := 429
				if mode == "capacity" {
					want = 503
				}
				if resp.StatusCode != want || resp.Header.Get("Retry-After") != "5" {
					t.Fatalf("quota status: %d %v", resp.StatusCode, resp.Header)
				}
				return
			}
			if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "text/event-stream; charset=utf-8" || resp.Header.Get("X-Accel-Buffering") != "no" {
				t.Fatalf("SSE headers: %d %v", resp.StatusCode, resp.Header)
			}
			scanner := bufio.NewScanner(resp.Body)
			for _, want := range []string{"event: ready", `data: {"status":"connected"}`, ""} {
				if !scanner.Scan() || scanner.Text() != want {
					t.Fatal("missing ready frame")
				}
			}
			m := domain.Message{Version: 1, ID: "test", Type: "ring_awarded", Occurred: now, Expires: time.Now().Add(20 * time.Second), Name: "昵称", ItemID: 1, ItemName: "戒指", Quantity: 1, Source: "normal_draw"}
			raw, _ := json.Marshal(m)
			h.Receive(raw)
			frame, _ := EncodeFrame(m)
			for _, want := range strings.Split(strings.TrimSuffix(frame, "\n"), "\n") {
				if !scanner.Scan() || scanner.Text() != want {
					t.Fatal("SSE frame changed")
				}
			}
			switch mode {
			case "close":
				h.Close()
			case "disconnect":
				cancel()
				resp.Body.Close()
			case "unavailable":
				h.SetReady(false)
			}
			if mode != "disconnect" {
				var tail []string
				for scanner.Scan() {
					tail = append(tail, scanner.Text())
				}
				if scanner.Err() != nil {
					t.Fatal(scanner.Err())
				}
				if mode == "unavailable" && !strings.Contains(strings.Join(tail, "\n"), "event: unavailable") {
					t.Fatal("missing unavailable event")
				}
			}
			until := time.Now().Add(3 * time.Second)
			for h.Stats().Connections != 0 {
				if time.Now().After(until) {
					t.Fatal("handler leaked connection")
				}
				runtime.Gosched()
			}
		})
	}
}
