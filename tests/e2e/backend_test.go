package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestBusinessFlow(t *testing.T) {
	base := os.Getenv("E2E_BASE_URL")
	if base == "" {
		t.Skip("set E2E_BASE_URL and E2E_MYSQL_DSN for isolated integration environment")
	}
	db, e := sql.Open("mysql", os.Getenv("E2E_MYSQL_DSN"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Timeout: 10 * time.Second, Jar: jar}
	call := func(path, token string, in any, status int) map[string]any {
		t.Helper()
		raw, _ := json.Marshal(in)
		req, _ := http.NewRequest("POST", base+path, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", base)
		req.Header.Set("X-Activity-CSRF", "1")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, e := client.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != status {
			t.Fatalf("%s: %d want %d: %s", path, resp.StatusCode, status, b)
		}
		out := map[string]any{}
		_ = json.Unmarshal(b, &out)
		return out
	}
	eventually := func(fn func() bool) {
		t.Helper()
		until := time.Now().Add(40 * time.Second)
		for time.Now().Before(until) {
			if fn() {
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		t.Fatal("eventual condition timed out")
	}
	register := func(suffix string) (string, string) {
		a := call("/api/identity/v1/auth/register", "", map[string]any{"account": fmt.Sprintf("t%d%s", time.Now().Unix()%1000000, suffix), "password": "test-secret-123"}, 200)
		return a["access_token"].(string), a["user_id"].(string)
	}
	token, user := register("a")
	eventually(func() bool {
		var n int
		_ = db.QueryRow(`SELECT COUNT(*) FROM wallet WHERE user_id=?`, user).Scan(&n)
		return n == 1
	})
	edition := os.Getenv("E2E_EDITION_ID")
	cmd := func(action string, v map[string]any, status int) map[string]any {
		v["edition_id"] = edition
		return call("/api/activity/v1/"+action, token, v, status)
	}
	joined := cmd("join", map[string]any{}, 200)
	if joined["gold"] != float64(10000000) || joined["petals"] != float64(100000000) {
		t.Fatal("incorrect initialization")
	}
	op := fmt.Sprint(time.Now().UnixNano())
	draw := cmd("draw", map[string]any{"operation_id": op, "pool_code": "NIGHT", "count": 1}, 200)
	if draw["flowers"] != float64(3) {
		t.Fatal("night first draw should light three flowers")
	}
	replay := cmd("draw", map[string]any{"operation_id": op, "pool_code": "NIGHT", "count": 1}, 200)
	if draw["draw_id"] != replay["draw_id"] {
		t.Fatal("replay changed draw")
	}
	cmd("draw", map[string]any{"operation_id": op, "pool_code": "DAY", "count": 1}, 409)
	cmd("stage", map[string]any{"stage": 3}, 200)
	cmd("stage", map[string]any{"stage": 3}, 200)
	purchase := map[string]any{"operation_id": "buy-" + op, "product_code": "52"}
	cmd("purchase", purchase, 200)
	cmd("purchase", purchase, 200)
	cmd("purchase", map[string]any{"operation_id": "other-" + op, "product_code": "52"}, 409)
	_, e = db.Exec(`UPDATE user_round SET flower_count=17,unlocked_chests=2,consumed_gold_equivalent=2082200 WHERE user_id=? AND edition_id=? AND round_no=1`, user, edition)
	if e != nil {
		t.Fatal(e)
	}
	filled := cmd("draw", map[string]any{"operation_id": "full-" + op, "pool_code": "DAY", "count": 30}, 200)
	if filled["actual_count"] != float64(1) || filled["refunded_petals"] != float64(145) {
		t.Fatalf("incorrect full stop: %v", filled)
	}
	for _, stage := range []int{5, 8, 11, 15, 17} {
		cmd("stage", map[string]any{"stage": stage}, 200)
	}
	for i := 0; i < 3; i++ {
		chest := cmd("chest-open", map[string]any{}, 200)
		same := cmd("chest-open", map[string]any{}, 200)
		if chest["chest_id"] != same["chest_id"] {
			t.Fatal("chest changed")
		}
		var status string
		_ = db.QueryRow(`SELECT status FROM reward_grant WHERE grant_id=?`, chest["grant_id"]).Scan(&status)
		if status == "PENDING_CHOICE" {
			cmd("select", map[string]any{"id": chest["grant_id"], "item_id": "1207751"}, 200)
		}
		cmd("chest-confirm", map[string]any{"id": chest["chest_id"]}, 200)
	}
	var stageGrant string
	_ = db.QueryRow(`SELECT CAST(grant_id AS CHAR) FROM reward_grant WHERE user_id=? AND edition_id=? AND source_type='STAGE' AND status='PENDING_CHOICE' LIMIT 1`, user, edition).Scan(&stageGrant)
	cmd("select", map[string]any{"id": stageGrant, "item_id": "1207752"}, 200)
	cmd("next-round", map[string]any{"from_round_no": 1}, 200)
	cmd("next-round", map[string]any{"from_round_no": 1}, 200)
	var shard int
	_ = db.QueryRow(`SELECT MOD(?,4)`, user).Scan(&shard)
	var details, flowers, rewards int
	_ = db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM draw_detail_%d WHERE user_id=?`, shard), user).Scan(&details)
	_ = db.QueryRow(fmt.Sprintf(`SELECT SUM(flowers_added) FROM flower_lit_log_%d WHERE user_id=?`, shard), user).Scan(&flowers)
	_ = db.QueryRow(`SELECT COUNT(*) FROM reward_grant WHERE user_id=? AND draw_id IS NOT NULL`, user).Scan(&rewards)
	if details != 2 || rewards != 2 || flowers != 4 {
		t.Fatalf("fact mismatch: details=%d grants=%d flowers=%d", details, rewards, flowers)
	}
	token2, user2 := register("b")
	eventually(func() bool {
		var n int
		_ = db.QueryRow(`SELECT COUNT(*) FROM wallet WHERE user_id=?`, user2).Scan(&n)
		return n == 1
	})
	token = token2
	preview := cmd("preview", map[string]any{}, 200)
	same := cmd("preview", map[string]any{}, 200)
	if preview["preview_id"] != same["preview_id"] {
		t.Fatal("preview changed")
	}
	if !reflect.DeepEqual(preview, same) || same["required_petals"] == nil {
		t.Fatal("preview recovery lost price or results")
	}
	confirmed := cmd("preview-confirm", map[string]any{}, 200)
	again := cmd("preview-confirm", map[string]any{}, 200)
	if confirmed["draw_id"] != again["draw_id"] {
		t.Fatal("preview duplicated")
	}
	if !reflect.DeepEqual(confirmed, again) || again["results"] == nil {
		t.Fatal("confirmation recovery lost complete result")
	}
	renewed := call("/api/identity/v1/auth/refresh", "", map[string]any{}, 200)
	if renewed["access_token"] == "" {
		t.Fatal("refresh missing access")
	}
	call("/api/identity/v1/auth/logout", "", map[string]any{}, 200)
	call("/api/activity/v1/join", renewed["access_token"].(string), map[string]any{"edition_id": edition}, 401)
	_ = context.Background()
}
