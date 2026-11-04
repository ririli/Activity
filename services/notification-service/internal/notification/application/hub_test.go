package application

import (
	"activity/pkg/platform"
	"context"
	"testing"
	"time"
)

func TestHubQuotasBackpressureAndCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := New(ctx)
	if _, e := h.Subscribe("sid"); e == nil {
		t.Fatal("accepted while unavailable")
	}
	h.SetReady(true)
	clients := []*Connection{}
	for i := 0; i < 5; i++ {
		c, e := h.Subscribe("sid")
		if e != nil {
			t.Fatal(e)
		}
		clients = append(clients, c)
	}
	if _, e := h.Subscribe("sid"); e == nil {
		t.Fatal("quota not enforced")
	}
	for i := 0; i < 100; i++ {
		h.Receive(platform.JSON(map[string]any{"schema_version": 1, "event_id": "ring:1:v1", "event_type": "ring_awarded", "occurred_at": time.Now(), "expires_at": time.Now().Add(time.Second), "source": "normal_draw", "display_name": "user", "item_id": 1207244, "item_name": "ring", "quantity": 1}))
	}
	for _, c := range clients {
		if len(c.Messages) != 32 {
			t.Fatal("queue not bounded")
		}
		h.Unsubscribe(c)
	}
	if len(h.clients) != 0 {
		t.Fatal("connection leak")
	}
}
