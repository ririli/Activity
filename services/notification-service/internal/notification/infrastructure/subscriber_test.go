package infrastructure

import (
	"activity/services/notification-service/internal/notification/application"
	"activity/services/notification-service/internal/notification/domain"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type pingHook struct{ count atomic.Int32 }

func (h *pingHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) { return next(ctx, network, addr) }
}
func (h *pingHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h *pingHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if cmd.Name() == "ping" {
			h.count.Add(1)
		}
		return next(ctx, cmd)
	}
}

func TestRedisBroadcastSubscription(t *testing.T) {
	addr := os.Getenv("NOTIFICATION_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set NOTIFICATION_TEST_REDIS_ADDR for isolated Redis integration")
	}
	env := fmt.Sprintf("notification-test-%d", time.Now().UnixNano())
	t.Setenv("APP_ENV", env)
	r := redis.NewClient(&redis.Options{Addr: addr, MaxRetries: -1, ReadTimeout: time.Second, WriteTimeout: time.Second})
	defer r.Close()
	hook := &pingHook{}
	r.AddHook(hook)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h, err := application.New(ctx, application.DefaultConfig(), func(m domain.Message) (string, error) { return m.ID, nil })
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	done := make(chan struct{})
	go func() { defer close(done); Subscribe(ctx, r, h) }()
	wait := func(check func() bool) {
		t.Helper()
		poll := time.NewTicker(time.Millisecond)
		defer poll.Stop()
		deadline := time.NewTimer(5 * time.Second)
		defer deadline.Stop()
		for !check() {
			select {
			case <-poll.C:
			case <-deadline.C:
				t.Fatal("subscription state timed out")
			}
		}
	}
	wait(h.Ready)
	c, err := h.Subscribe("sid")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		now := time.Now()
		raw, _ := json.Marshal(domain.Message{Version: 1, ID: fmt.Sprint(i), Type: "ring_awarded", Occurred: now, Expires: now.Add(30 * time.Second), Source: "normal_draw", Name: "user", ItemID: 1, ItemName: "ring", Quantity: 1})
		if err = r.Publish(ctx, "activity:"+env+":notification:ring-awarded:v1", raw).Err(); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 20; i++ {
		select {
		case f := <-c.Messages:
			if f.Data != fmt.Sprint(i) {
				t.Fatal("subscription order changed")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Redis delivery timed out")
		}
	}
	if n := hook.count.Load(); n >= 20 {
		t.Fatalf("PING still executed per message: %d", n)
	}
	_ = r.Close()
	wait(func() bool { return !h.Ready() })
	if _, err = h.Subscribe("offline"); err != application.ErrUnavailable {
		t.Fatal("accepted while unavailable")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("subscriber leaked")
	}
}
