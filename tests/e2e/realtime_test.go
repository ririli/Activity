package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestProjectionAndSSE(t *testing.T) {
	base := os.Getenv("E2E_BASE_URL")
	addr := os.Getenv("E2E_REDIS_ADDR")
	if base == "" || addr == "" {
		t.Skip("requires isolated HTTP, Redis, and seeded active edition")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	register := map[string]string{"account": fmt.Sprintf("rt%d", time.Now().Unix()%100000000), "password": "test-secret-123", "nickname": "实时测试"}
	raw, _ := json.Marshal(register)
	resp, e := client.Post(base+"/api/identity/v1/auth/register", "application/json", bytes.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	var auth struct {
		Token string `json:"access_token"`
		User  string `json:"user_id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&auth)
	resp.Body.Close()
	if auth.Token == "" {
		t.Fatal("registration failed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	edition := os.Getenv("E2E_EDITION_ID")
	for {
		req, _ := http.NewRequestWithContext(ctx, "GET", base+"/api/ranking/v1/board?edition_id="+edition, nil)
		req.Header.Set("Authorization", "Bearer "+auth.Token)
		r, e := client.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		var board struct {
			Me *struct {
				User     string `json:"user_id"`
				Score    int64  `json:"score"`
				Nickname string `json:"nickname"`
			} `json:"me"`
		}
		_ = json.NewDecoder(r.Body).Decode(&board)
		r.Body.Close()
		if board.Me != nil && board.Me.User == auth.User {
			if board.Me.Score != 0 || board.Me.Nickname != "实时测试" {
				t.Fatal("invalid zero-score projection/profile")
			}
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("zero score initialization timed out")
		case <-time.After(time.Second):
		}
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", base+"/api/notification/v1/barrage/stream", nil)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	stream, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer stream.Body.Close()
	if stream.StatusCode != 200 {
		t.Fatalf("SSE returned %d", stream.StatusCode)
	}
	scanner := bufio.NewScanner(stream.Body)
	if !scanner.Scan() || scanner.Text() != "event: ready" {
		t.Fatal("SSE not ready")
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	defer rdb.Close()
	event := map[string]any{"schema_version": 1, "event_type": "ring_awarded", "event_id": "ring:e2e:1:v1", "occurred_at": time.Now().UTC(), "expires_at": time.Now().Add(20 * time.Second).UTC(), "edition_id": edition, "source": "normal_draw", "display_name": "实时测试", "item_id": 1207244, "item_name": "烟花之恋戒指", "quantity": 1}
	b, _ := json.Marshal(event)
	if e = rdb.Publish(ctx, "activity:local:notification:ring-awarded:v1", b).Err(); e != nil {
		t.Fatal(e)
	}
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "ring:e2e:1:v1") {
			return
		}
	}
	t.Fatalf("broadcast not delivered: %v", scanner.Err())
}
