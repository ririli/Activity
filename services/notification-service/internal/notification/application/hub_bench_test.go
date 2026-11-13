package application

import (
	"activity/services/notification-service/internal/notification/domain"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"
)

// This is the pre-sharding algorithm, including per-recipient JSON encoding in
// its simulated HTTP readers. It intentionally permits 10000 synthetic clients
// to compare the algorithm beyond the former 1000-connection admission limit.
type legacyHub struct {
	mu      sync.Mutex
	clients map[chan domain.Message]bool
}

func (h *legacyHub) receive(raw []byte) {
	m, err := domain.Parse(raw, time.Now())
	if err != nil {
		panic(err)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c <- m:
		default:
		}
	}
}
func benchEncode(m domain.Message) (string, error) {
	b, err := json.Marshal(map[string]any{"message_id": m.ID, "display_name": m.Name, "item_name": m.ItemName, "quantity": m.Quantity, "expires_at": m.Expires})
	return "event: barrage\nid: " + m.ID + "\ndata: " + string(b) + "\n\n", err
}

func BenchmarkBroadcastCompletion(b *testing.B) {
	for _, n := range []int{1000, 10000} {
		for _, mode := range []string{"legacy", "shards1", "shards8", "shards8_skewed"} {
			b.Run(fmt.Sprintf("%s/connections%d", mode, n), func(b *testing.B) {
				b.StopTimer()
				stop := make(chan struct{})
				ack := make(chan struct{}, n)
				var readers sync.WaitGroup
				var publish func([]byte)
				var hub *Hub
				if mode == "legacy" {
					h := &legacyHub{clients: make(map[chan domain.Message]bool)}
					for i := 0; i < n; i++ {
						q := make(chan domain.Message, 32)
						h.clients[q] = true
						readers.Add(1)
						go func() {
							defer readers.Done()
							for {
								select {
								case <-stop:
									return
								case m := <-q:
									_, _ = benchEncode(m)
									ack <- struct{}{}
								}
							}
						}()
					}
					publish = h.receive
				} else {
					cfg := DefaultConfig()
					if mode == "shards1" {
						cfg.Shards = 1
					}
					var err error
					hub, err = New(b.Context(), cfg, benchEncode)
					if err != nil {
						b.Fatal(err)
					}
					b.Cleanup(hub.Close)
					hub.SetReady(true)
					for i := 0; i < n; i++ {
						c := subscribe(b, hub, fmt.Sprint(i))
						if mode == "shards8_skewed" { // model disconnect churn leaving one hot shard
							s := hub.shards[c.shard]
							s.mu.Lock()
							delete(s.clients, c)
							s.mu.Unlock()
							c.shard = 0
							s = hub.shards[0]
							s.mu.Lock()
							s.clients[c] = struct{}{}
							s.mu.Unlock()
						}
						readers.Add(1)
						go func() {
							defer readers.Done()
							for {
								select {
								case <-stop:
									return
								case <-c.Messages:
									ack <- struct{}{}
								}
							}
						}()
					}
					publish = hub.Receive
				}
				defer func() { close(stop); readers.Wait() }()
				raw := testRaw("benchmark")
				samples := make([]time.Duration, 0, b.N)
				deadline := time.NewTimer(2 * time.Minute)
				defer deadline.Stop()
				b.ResetTimer()
				b.StartTimer()
				for i := 0; i < b.N; i++ {
					started := time.Now()
					publish(raw)
					for j := 0; j < n; j++ {
						select {
						case <-ack:
						case <-deadline.C:
							b.Fatal("broadcast lost or expired; use bounded -benchtime")
						}
					}
					samples = append(samples, time.Since(started))
				}
				b.StopTimer()
				sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
				if len(samples) > 0 {
					for _, q := range []int{50, 95, 99} {
						b.ReportMetric(float64(samples[(len(samples)*q+99)/100-1].Microseconds()), fmt.Sprintf("p%d-us", q))
					}
				}
				b.ReportMetric(float64(n), "recipients/op")
				if hub != nil {
					s := hub.Stats()
					b.ReportMetric(float64(s.ShardDropped+s.ConnectionDropped), "drops")
				}
			})
		}
	}
}
