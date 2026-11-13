package http

import (
	"activity/services/notification-service/internal/notification/domain"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestEncodeFrameContract(t *testing.T) {
	m := domain.Message{ID: "ring:1:v1", Name: "昵称\n<script>", ItemName: "戒指", Quantity: 2, Expires: time.Now().UTC()}
	f, err := EncodeFrame(m)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(f, "\n")
	if len(lines) != 5 || lines[0] != "event: barrage" || lines[1] != "id: ring:1:v1" || lines[3] != "" || lines[4] != "" {
		t.Fatal(f)
	}
	var payload map[string]any
	if err = json.Unmarshal([]byte(strings.TrimPrefix(lines[2], "data: ")), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) != 5 || payload["message_id"] != m.ID || payload["display_name"] != m.Name || payload["item_name"] != m.ItemName || payload["quantity"] != float64(2) || payload["expires_at"] != m.Expires.Format(time.RFC3339Nano) {
		t.Fatalf("contract changed: %v", payload)
	}
}
