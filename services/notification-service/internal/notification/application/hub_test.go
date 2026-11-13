package application

import (
	"activity/services/notification-service/internal/notification/domain"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testRaw(id string) []byte {
	now := time.Now()
	b, _ := json.Marshal(domain.Message{Version: 1, ID: id, Type: "ring_awarded", Occurred: now, Expires: now.Add(30 * time.Second), Source: "normal_draw", Name: "user", ItemID: 1207244, ItemName: "ring", Quantity: 1})
	return b
}
func testHub(t testing.TB, cfg Config) *Hub {
	t.Helper()
	h, err := New(context.Background(), cfg, func(m domain.Message) (string, error) { return m.ID, nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)
	h.SetReady(true)
	return h
}
func subscribe(t testing.TB, h *Hub, session string) *Connection {
	t.Helper()
	c, err := h.Subscribe(session)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func receive(t testing.TB, c *Connection) Frame {
	t.Helper()
	select {
	case f := <-c.Messages:
		return f
	case <-time.After(5 * time.Second):
		t.Fatal("delivery timed out")
		return Frame{}
	}
}
func eventually(t testing.TB, f func() bool) {
	t.Helper()
	until := time.Now().Add(5 * time.Second)
	for !f() {
		if time.Now().After(until) {
			t.Fatal("condition timed out")
		}
		runtime.Gosched()
	}
}

func TestHubConfig(t *testing.T) {
	for field := 0; field < 5; field++ {
		c := DefaultConfig()
		fields := []*int{&c.Shards, &c.MaxConnections, &c.SessionLimit, &c.ShardQueue, &c.ConnectionQueue}
		*fields[field] = 0
		if _, err := New(context.Background(), c, func(domain.Message) (string, error) { return "", nil }); err == nil {
			t.Fatal("accepted zero config")
		}
	}
	if _, err := New(context.Background(), DefaultConfig(), nil); err == nil {
		t.Fatal("accepted nil encoder")
	}
}
func TestHubRoundRobinAndInstanceQuotas(t *testing.T) {
	h := testHub(t, DefaultConfig())
	h.SetReady(false)
	if _, err := h.Subscribe("sid"); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	h.SetReady(true)
	var first *Connection
	for i := 0; i < 5; i++ {
		c := subscribe(t, h, "sid")
		if c.shard != i {
			t.Fatal("not round robin")
		}
		if i == 0 {
			first = c
		}
	}
	if _, err := h.Subscribe("sid"); !errors.Is(err, ErrSessionLimit) {
		t.Fatal(err)
	}
	h.Unsubscribe(first)
	h.Unsubscribe(first)
	subscribe(t, h, "sid")
	for i := 5; i < 10000; i++ {
		subscribe(t, h, fmt.Sprint(i))
	}
	if _, err := h.Subscribe("extra"); !errors.Is(err, ErrCapacity) {
		t.Fatal(err)
	}
	for _, s := range h.shards {
		s.mu.Lock()
		n := len(s.clients)
		s.mu.Unlock()
		if n != 1250 {
			t.Fatalf("unbalanced shard: %d", n)
		}
	}
	h.Close()
	h.Close()
	if h.Ready() || h.Stats().Connections != 0 || len(h.sessionCounts) != 0 {
		t.Fatal("close leaked quota")
	}
	for _, s := range h.shards {
		if len(s.clients) != 0 || len(s.queue) != 0 {
			t.Fatal("close leaked references")
		}
	}
	h.SetReady(true)
	if _, err := h.Subscribe("closed"); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
}
func TestHubBroadcastOrderAndEncodeOnce(t *testing.T) {
	var encoded atomic.Int32
	h, err := New(context.Background(), DefaultConfig(), func(m domain.Message) (string, error) { encoded.Add(1); return m.ID, nil })
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	h.SetReady(true)
	clients := make([]*Connection, 16)
	for i := range clients {
		clients[i] = subscribe(t, h, fmt.Sprint(i))
	}
	for i := 0; i < 20; i++ {
		h.Receive(testRaw(fmt.Sprint(i)))
	}
	for _, c := range clients {
		for i := 0; i < 20; i++ {
			if f := receive(t, c); f.Data != fmt.Sprint(i) {
				t.Fatalf("out of order: %s", f.Data)
			}
		}
		select {
		case <-c.Messages:
			t.Fatal("duplicate")
		default:
		}
	}
	if encoded.Load() != 20 {
		t.Fatalf("encoded %d times", encoded.Load())
	}
}
func TestHubBlockedShardDoesNotBlockIngressOrPeers(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ShardQueue = 1
	h := testHub(t, cfg)
	clients := make([]*Connection, 8)
	for i := range clients {
		clients[i] = subscribe(t, h, fmt.Sprint(i))
	}
	blocked := h.shards[0]
	blocked.mu.Lock()
	var unlock sync.Once
	defer unlock.Do(blocked.mu.Unlock)
	h.Receive(testRaw("first"))
	eventually(t, func() bool { return len(blocked.queue) == 0 }) // worker dequeued and is waiting on its lock
	h.Receive(testRaw("queued"))
	returned := make(chan struct{})
	go func() { h.Receive(testRaw("overflow")); close(returned) }()
	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		t.Fatal("ingress waited for shard")
	}
	for _, c := range clients[1:] {
		if receive(t, c).Data != "first" {
			t.Fatal("peer stalled")
		}
	}
	if h.Stats().ShardDropped < 1 {
		t.Fatal("full shard did not drop")
	}
	unlock.Do(blocked.mu.Unlock)
	if receive(t, clients[0]).Data != "first" || receive(t, clients[0]).Data != "queued" {
		t.Fatal("blocked shard lost order")
	}
	select {
	case <-clients[0].Messages:
		t.Fatal("overflow delivered")
	default:
	}
}
func TestHubConnectionBackpressureAndExpiry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Shards = 1
	cfg.ConnectionQueue = 1
	h := testHub(t, cfg)
	slow := subscribe(t, h, "slow")
	fast := subscribe(t, h, "fast")
	h.Receive(testRaw("first"))
	receive(t, fast)
	h.Receive(testRaw("second"))
	if receive(t, fast).Data != "second" {
		t.Fatal("slow client stalled fast client")
	}
	eventually(t, func() bool { return h.Stats().ConnectionDropped == 1 })
	if receive(t, slow).Data != "first" {
		t.Fatal("full queue overwrote old message")
	}
	h.Receive([]byte("invalid"))
	h.Receive(make([]byte, 4097))
	h.shards[0].queue <- Frame{Data: "expired", Expires: time.Now().Add(-time.Second)}
	h.Receive(testRaw("barrier"))
	if receive(t, fast).Data != "barrier" || receive(t, slow).Data != "barrier" {
		t.Fatal("expired message delivered")
	}
	if s := h.Stats(); s.Expired != 1 || s.Invalid != 2 {
		t.Fatalf("stats: %+v", s)
	}
	h.Unsubscribe(slow)
	h.Unsubscribe(slow)
	h.Unsubscribe(fast)
	if h.Stats().Connections != 0 || len(h.sessionCounts) != 0 {
		t.Fatal("unsubscribe leaked quota")
	}
}
func TestHubExpiryAfterWaitingForShardLock(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Shards = 1
	h := testHub(t, cfg)
	c := subscribe(t, h, "sid")
	s := h.shards[0]
	s.mu.Lock()
	var unlock sync.Once
	defer unlock.Do(s.mu.Unlock)
	// Queue an already expired frame to exercise worker-side checking independently of parsing.
	s.queue <- Frame{Data: "expired", Expires: time.Now().Add(-time.Second)}
	eventually(t, func() bool { return len(s.queue) == 0 })
	h.Receive(testRaw("barrier"))
	unlock.Do(s.mu.Unlock)
	if receive(t, c).Data != "barrier" {
		t.Fatal("worker forwarded expired frame")
	}
}
func TestHubEncoderFailure(t *testing.T) {
	h, err := New(context.Background(), DefaultConfig(), func(domain.Message) (string, error) { return "", errors.New("encode") })
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	h.SetReady(true)
	h.Receive(testRaw("x"))
	if h.Stats().Invalid != 1 {
		t.Fatal("encoding error not dropped")
	}
}
func TestHubConcurrentLifecycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h, err := New(ctx, DefaultConfig(), func(m domain.Message) (string, error) { return m.ID, nil })
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	h.SetReady(true)
	raw := testRaw("race")
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			for j := 0; j < 100; j++ {
				c, e := h.Subscribe(fmt.Sprint(i))
				if e == nil {
					h.Receive(raw)
					h.Unsubscribe(c)
					h.Unsubscribe(c)
				}
				if j == 50 {
					h.Close()
				}
			}
		}(i)
	}
	close(start)
	done := make(chan struct{})
	go func() { wg.Wait(); cancel(); h.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("lifecycle deadlock")
	}
	if h.Stats().Connections != 0 || len(h.sessionCounts) != 0 {
		t.Fatal("lifecycle leaked connections")
	}
	select {
	case <-h.Done():
	default:
		t.Fatal("Done not canceled")
	}
}

func TestHubConcurrentSessionQuota(t *testing.T) {
	h := testHub(t, DefaultConfig())
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := h.Subscribe("same-session"); err == nil {
				accepted.Add(1)
			} else if !errors.Is(err, ErrSessionLimit) {
				t.Errorf("unexpected quota error: %v", err)
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 5 || h.Stats().Connections != 5 {
		t.Fatal("cross-shard quota overshoot")
	}
}

func TestHubParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h, err := New(ctx, DefaultConfig(), func(m domain.Message) (string, error) { return m.ID, nil })
	if err != nil {
		t.Fatal(err)
	}
	h.SetReady(true)
	subscribe(t, h, "sid")
	h.Receive(testRaw("pending"))
	cancel()
	select {
	case <-h.Done():
	case <-time.After(time.Second):
		t.Fatal("parent cancellation not propagated")
	}
	h.Close()
	if h.Ready() || h.Stats().Connections != 0 {
		t.Fatal("parent cancellation leaked state")
	}
}
