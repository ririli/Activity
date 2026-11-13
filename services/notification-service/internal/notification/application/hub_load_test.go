package application

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Opt-in sustained fanout test. Readers continuously consume full encoded
// frames. This measures in-memory delivery, not sockets, authentication or RSS.
func TestHubSustainedLoad(t *testing.T) {
	if os.Getenv("NOTIFICATION_HUB_LOAD") != "1" {
		t.Skip("set NOTIFICATION_HUB_LOAD=1 for sustained synthetic load")
	}
	duration := 2 * time.Second
	if raw := os.Getenv("NOTIFICATION_HUB_LOAD_DURATION"); raw != "" {
		var err error
		duration, err = time.ParseDuration(raw)
		if err != nil || duration <= 0 || duration > 20*time.Second {
			t.Fatal("load duration must be positive and at most 20s")
		}
	}
	for _, n := range []int{1000, 10000} {
		for _, shards := range []int{1, 8} {
			for _, rate := range []int{1, 10, 100} {
				t.Run(fmt.Sprintf("connections%d/shards%d/rate%d/cpu%d", n, shards, rate, runtime.GOMAXPROCS(0)), func(t *testing.T) {
					cfg := DefaultConfig()
					cfg.Shards = shards
					h, err := New(t.Context(), cfg, benchEncode)
					if err != nil {
						t.Fatal(err)
					}
					defer h.Close()
					h.SetReady(true)
					ctx, cancel := context.WithCancel(t.Context())
					var readers sync.WaitGroup
					var delivered atomic.Uint64
					var histogram [1001]atomic.Uint64
					const burst = 64
					starts := make([]atomic.Pointer[time.Time], int(duration.Seconds()*float64(rate))+burst+10)
					clients := make([]*Connection, n)
					registrationStarted := time.Now()
					for i := range clients {
						clients[i] = subscribe(t, h, fmt.Sprint(i))
					}
					t.Logf("register_all=%s", time.Since(registrationStarted))
					for _, c := range clients {
						readers.Add(1)
						go func() {
							defer readers.Done()
							for {
								select {
								case <-ctx.Done():
									return
								case f := <-c.Messages:
									id := strings.TrimPrefix(f.Data, "event: barrage\nid: ")
									end := strings.IndexByte(id, '\n')
									if end < 0 {
										continue
									}
									index, err := strconv.Atoi(id[:end])
									if err != nil || index >= len(starts) {
										continue
									}
									if start := starts[index].Load(); start != nil {
										ms := int64((time.Since(*start) + time.Millisecond - 1) / time.Millisecond)
										if ms < 0 {
											ms = 0
										}
										if ms > 1000 {
											ms = 1000
										}
										histogram[ms].Add(1)
										delivered.Add(1)
									}
								}
							}
						}()
					}
					defer func() { cancel(); readers.Wait() }()
					var before, after runtime.MemStats
					runtime.ReadMemStats(&before)
					sent := 0
					publish := func() { now := time.Now(); starts[sent].Store(&now); h.Receive(testRaw(strconv.Itoa(sent))); sent++ }
					started := time.Now()
					publish()
					pace := time.NewTicker(time.Second / time.Duration(rate))
					defer pace.Stop()
					end := time.NewTimer(duration)
					defer end.Stop()
				loop:
					for {
						select {
						case <-end.C:
							break loop
						case <-pace.C:
							publish()
						}
					}
					accounted := func() bool {
						s := h.Stats()
						return delivered.Load()+s.ConnectionDropped+s.ShardDropped*uint64(n/shards)+s.Expired*uint64(n/shards) == uint64(sent*n)
					}
					eventually(t, accounted)
					steady := h.Stats()
					t.Logf("steady_messages=%d steady_deliveries=%d steady_shard_drops=%d steady_connection_drops=%d", sent, delivered.Load(), steady.ShardDropped, steady.ConnectionDropped)
					for i := 0; i < burst; i++ {
						publish()
					}
					eventually(t, accounted)
					runtime.ReadMemStats(&after)
					quantile := func(q uint64) int {
						want := (delivered.Load()*q + 99) / 100
						var count uint64
						for i := range histogram {
							count += histogram[i].Load()
							if count >= want {
								return i
							}
						}
						return 1000
					}
					s := h.Stats()
					t.Logf("elapsed=%s sent=%d deliveries=%d p50_ms_upper=%d p95_ms_upper=%d p99_ms_upper=%d shard_drops=%d connection_drops=%d heap_bytes=%d go_sys_bytes=%d allocated_bytes=%d gc_cycles=%d gc_pause_ns=%d", time.Since(started), sent, delivered.Load(), quantile(50), quantile(95), quantile(99), s.ShardDropped, s.ConnectionDropped, after.HeapAlloc, after.Sys, after.TotalAlloc-before.TotalAlloc, after.NumGC-before.NumGC, after.PauseTotalNs-before.PauseTotalNs)
					if s.Connections != n {
						t.Fatal("load lost admitted connections")
					}
					start := time.Now()
					for _, c := range clients {
						h.Unsubscribe(c)
					}
					t.Logf("unregister_all=%s", time.Since(start))
				})
			}
		}
	}
}
