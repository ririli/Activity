// sseload measures real SSE delivery through the configured HTTP endpoint.
// Tokens must be created in an isolated load environment; this tool never
// bypasses authentication or changes service/proxy limits.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	url := flag.String("url", "http://localhost:8080/api/notification/v1/barrage/stream", "real SSE endpoint")
	tokensFile := flag.String("tokens", "", "JSON array of Access tokens; never printed")
	n := flag.Int("connections", 10000, "concurrent SSE connections")
	perToken := flag.Int("per-token", 5, "connections per distinct session/token")
	ramp := flag.Int("connect-rate", 20, "new connections per second (use isolated proxy limits)")
	rate := flag.Int("rate", 10, "broadcast messages per second")
	burst := flag.Int("burst", 0, "additional messages sent immediately after all streams are ready")
	duration := flag.Duration("duration", time.Minute, "broadcast duration after all streams are ready")
	settle := flag.Duration("settle", 2*time.Second, "final delivery drain time")
	flag.Parse()
	if *n <= 0 || *perToken <= 0 || *ramp <= 0 || *rate <= 0 || *rate > 10000 || *ramp > 10000 || *burst < 0 || *duration <= 0 || *settle < 0 {
		return fmt.Errorf("invalid load parameters")
	}
	b, err := os.ReadFile(*tokensFile)
	if err != nil {
		return err
	}
	var tokens []string
	if err = json.Unmarshal(b, &tokens); err != nil {
		return fmt.Errorf("tokens must be a JSON string array")
	}
	seen := make(map[string]bool)
	for _, token := range tokens {
		if token == "" || seen[token] {
			return fmt.Errorf("empty or duplicate token")
		}
		seen[token] = true
	}
	if len(tokens) < (*n+*perToken-1) / *perToken {
		return fmt.Errorf("insufficient sessions: need at least %d tokens", (*n+*perToken-1) / *perToken)
	}
	addr := os.Getenv("SSE_LOAD_REDIS_ADDR")
	if addr == "" {
		return fmt.Errorf("set SSE_LOAD_REDIS_ADDR for the isolated publisher")
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("SSE_LOAD_REDIS_PASSWORD"), MaxRetries: -1})
	defer rdb.Close()
	appEnv := os.Getenv("SSE_LOAD_APP_ENV")
	if appEnv == "" {
		appEnv = "local"
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*n)*time.Second/time.Duration(*ramp)+30*time.Second+*duration+*settle)
	defer cancel()
	transport := &http.Transport{MaxIdleConns: *n, MaxConnsPerHost: *n, ResponseHeaderTimeout: 10 * time.Second}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}
	var readers sync.WaitGroup
	// Millisecond histogram bounds memory regardless of message or connection count.
	histogram := make([]atomic.Uint64, 60001)
	var delivered, streamFailures, publishFailures atomic.Uint64
	var starts sync.Map
	ready := make(chan error, *n)
	runID := fmt.Sprintf("load:%d:", time.Now().UnixNano())
	reader := func(token string) {
		defer readers.Done()
		req, _ := http.NewRequestWithContext(ctx, "GET", *url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := client.Do(req)
		if err != nil {
			ready <- fmt.Errorf("stream connection failed")
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			ready <- fmt.Errorf("stream returned HTTP %d", resp.StatusCode)
			return
		}
		scanner := bufio.NewScanner(resp.Body)
		announced := false
		event, id := "", ""
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case strings.HasPrefix(line, "event: "):
				event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "id: "):
				id = strings.TrimPrefix(line, "id: ")
			case line == "":
				if event == "ready" && !announced {
					announced = true
					ready <- nil
				}
				if event == "barrage" && strings.HasPrefix(id, runID) {
					if v, ok := starts.Load(id); ok {
						ms := int64((time.Since(v.(time.Time)) + time.Millisecond - 1) / time.Millisecond)
						if ms < 0 {
							ms = 0
						}
						if ms > 60000 {
							ms = 60000
						}
						histogram[ms].Add(1)
						delivered.Add(1)
					}
				}
				event, id = "", ""
			}
		}
		if !announced {
			ready <- fmt.Errorf("stream ended before ready")
		} else if ctx.Err() == nil {
			streamFailures.Add(1)
		}
	}
	tick := time.NewTicker(time.Second / time.Duration(*ramp))
	defer tick.Stop()
	for i := 0; i < *n; i++ {
		<-tick.C
		readers.Add(1)
		go reader(tokens[i / *perToken])
	}
	var admissionErr error
	for i := 0; i < *n; i++ {
		if e := <-ready; e != nil && admissionErr == nil {
			admissionErr = e
		}
	}
	if admissionErr != nil {
		cancel()
		readers.Wait()
		return admissionErr
	}
	fmt.Fprintf(os.Stderr, "%d SSE streams ready; starting broadcasts\n", *n)
	started := time.Now()
	sent := 0
	publish := func() {
		now := time.Now()
		id := fmt.Sprintf("%s%d", runID, sent)
		sent++
		starts.Store(id, now)
		payload, _ := json.Marshal(map[string]any{"schema_version": 1, "event_id": id, "event_type": "ring_awarded", "occurred_at": now.UTC(), "expires_at": now.Add(30 * time.Second).UTC(), "source": "normal_draw", "display_name": "SSE压测", "item_id": 1207244, "item_name": "测试戒指", "quantity": 1})
		op, stop := context.WithTimeout(ctx, 200*time.Millisecond)
		if err := rdb.Publish(op, "activity:"+appEnv+":notification:ring-awarded:v1", payload).Err(); err != nil {
			publishFailures.Add(1)
		}
		stop()
	}
	for i := 0; i < *burst; i++ {
		publish()
	}
	pace := time.NewTicker(time.Second / time.Duration(*rate))
	defer pace.Stop()
	end := time.NewTimer(*duration)
	defer end.Stop()
publishLoop:
	for {
		select {
		case <-end.C:
			break publishLoop
		case <-pace.C:
			publish()
		}
	}
	actualDuration := time.Since(started)
	drain := time.NewTimer(*settle)
	<-drain.C
	cancel()
	readers.Wait()
	count := delivered.Load()
	quantile := func(q uint64) int {
		if count == 0 {
			return -1
		}
		want := (count*q + 99) / 100
		var sum uint64
		for i := range histogram {
			sum += histogram[i].Load()
			if sum >= want {
				return i
			}
		}
		return 60000
	}
	// Stable field ordering makes results convenient to compare across runs.
	result := map[string]any{"connections": *n, "messages": sent, "deliveries": count, "expected_deliveries": uint64(sent) * uint64(*n), "stream_failures": streamFailures.Load(), "publish_failures": publishFailures.Load(), "publish_duration_sec": actualDuration.Seconds(), "p50_ms_upper": quantile(50), "p95_ms_upper": quantile(95), "p99_ms_upper": quantile(99)}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return err
	}
	if publishFailures.Load() > 0 || streamFailures.Load() > 0 {
		return fmt.Errorf("load completed with publisher or stream failures; inspect summary")
	}
	return nil
}
