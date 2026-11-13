package entrylimit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// Uses the real proxy configuration and an isolated HTTP upstream; no business
// database, keys or running application stack is required.
func TestNginxAdmission(t *testing.T) {
	if os.Getenv("RATE_LIMIT_TEST_NGINX") != "1" {
		t.Skip("set RATE_LIMIT_TEST_NGINX=1 to run isolated Docker proxy tests")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "upstream.go")
	binary := filepath.Join(dir, "upstream")
	if err := os.WriteFile(source, []byte(upstreamSource), 0600); err != nil {
		t.Fatal(err)
	}
	goBin := filepath.Join(runtime.GOROOT(), "bin", "go")
	if runtime.GOOS == "windows" {
		goBin += ".exe"
	}
	build := exec.Command(goBin, "build", "-o", binary, source)
	build.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build upstream: %s %v", out, err)
	}
	config, err := os.ReadFile("../../deploy/local/nginx/default.conf")
	if err != nil {
		t.Fatal(err)
	}
	baseConfig := string(config)
	for _, name := range []string{"identity", "activity", "ranking", "notification"} {
		baseConfig = strings.ReplaceAll(baseConfig, "http://"+name+":8080", "http://127.0.0.1:8080")
	}
	client := &http.Client{Timeout: 4 * time.Second}
	start := func(t *testing.T, config string) (string, string) {
		t.Helper()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "default.conf"), []byte(config), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("static-ok"), 0600); err != nil {
			t.Fatal(err)
		}
		id, err := exec.Command("docker", "run", "--rm", "-d", "-p", "127.0.0.1::80", "-v", filepath.Join(dir, "default.conf")+":/etc/nginx/conf.d/default.conf:ro", "-v", filepath.Join(dir, "index.html")+":/usr/share/nginx/html/index.html:ro", "-v", binary+":/verify/upstream:ro", "nginx:1.28-alpine", "sh", "-c", "/verify/upstream & exec nginx -g 'daemon off;'").CombinedOutput()
		if err != nil {
			t.Fatalf("docker: %s %v", id, err)
		}
		name := strings.TrimSpace(string(id))
		t.Cleanup(func() { exec.Command("docker", "rm", "-f", name).Run() })
		if out, err := exec.Command("docker", "exec", name, "nginx", "-t").CombinedOutput(); err != nil {
			t.Fatalf("nginx -t: %s %v", out, err)
		}
		out, err := exec.Command("docker", "port", name, "80").Output()
		if err != nil {
			t.Fatal(err)
		}
		base := "http://" + strings.TrimSpace(string(out))
		for i := 0; i < 30; i++ {
			resp, e := client.Get(base + "/")
			if e == nil {
				resp.Body.Close()
				return base, name
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Fatal("proxy did not start")
		return "", ""
	}
	get := func(t *testing.T, base, path string) (int, string, http.Header) {
		t.Helper()
		req, _ := http.NewRequest("GET", base+path, nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.99")
		req.Header.Set("X-Real-IP", "203.0.113.98")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b), resp.Header
	}
	t.Run("headers_and_backend_errors", func(t *testing.T) {
		base, _ := start(t, baseConfig)
		for _, svc := range []string{"identity", "activity", "ranking", "notification"} {
			status, body, _ := get(t, base, "/api/"+svc+"/echo")
			var v map[string]string
			json.Unmarshal([]byte(body), &v)
			if status != 200 || v["xff"] == "203.0.113.99" || v["real"] != v["xff"] || v["real"] == "" {
				t.Fatalf("headers: %d %s", status, body)
			}
		}
		status, body, header := get(t, base, "/api/identity/upstream429")
		if status != 429 || body != `{"error":"LOGIN_LIMITED"}` || header.Get("Retry-After") != "17" {
			t.Fatalf("upstream response overwritten: %d %s", status, body)
		}
		for _, path := range []string{"/internal/identity/v1/introspect", "/health/ready"} {
			if status, _, _ := get(t, base, path); status != 404 {
				t.Fatalf("exposed %s: %d", path, status)
			}
		}
	})
	for _, tc := range []struct{ name, config, path, zone string }{
		{"api_total", baseConfig, "/api/activity/echo", "api_total"},
		{"api_ip", strings.ReplaceAll(baseConfig, "rate=100r/s", "rate=10000r/s"), "/api/activity/echo", "api_ip"},
		{"auth", baseConfig, "/api/identity/v1/auth/login", "auth_ip"},
		{"sse_rate", baseConfig, "/api/notification/v1/barrage/stream", "sse_ip"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base, name := start(t, tc.config)
			var wg sync.WaitGroup
			var mu sync.Mutex
			rejected := 0
			for i := 0; i < 100; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					req, _ := http.NewRequestWithContext(ctx, "GET", base+tc.path, nil)
					resp, e := client.Do(req)
					if e != nil {
						t.Error(e)
						return
					}
					defer resp.Body.Close()
					if resp.StatusCode == 429 {
						b, _ := io.ReadAll(resp.Body)
						if string(b) != `{"error":"RATE_LIMITED"}` || resp.Header.Get("Retry-After") == "" {
							t.Errorf("rejection %s %v", b, resp.Header)
						}
						mu.Lock()
						rejected++
						mu.Unlock()
					}
				}()
			}
			wg.Wait()
			if rejected == 0 {
				t.Fatal("no rejections")
			}
			logs, _ := exec.Command("docker", "logs", name).CombinedOutput()
			if !strings.Contains(string(logs), `by zone "`+tc.zone+`"`) {
				t.Fatalf("expected zone %s in logs", tc.zone)
			}
			if status, body, _ := get(t, base, "/"); status != 200 || body != "static-ok" {
				t.Fatalf("static consumed API quota: %d %s", status, body)
			}
		})
	}
	t.Run("sse_capacity_and_release", func(t *testing.T) {
		base, _ := start(t, strings.ReplaceAll(baseConfig, "rate=1r/s", "rate=10000r/s"))
		streamClient := &http.Client{}
		var responses []*http.Response
		var cancels []context.CancelFunc
		defer func() {
			for _, cancel := range cancels {
				cancel()
			}
			for _, r := range responses {
				r.Body.Close()
			}
		}()
		for i := 0; i < 20; i++ {
			ctx, cancel := context.WithCancel(context.Background())
			cancels = append(cancels, cancel)
			req, _ := http.NewRequestWithContext(ctx, "GET", base+"/api/notification/v1/barrage/stream", nil)
			r, e := streamClient.Do(req)
			if e != nil {
				t.Fatal(e)
			}
			responses = append(responses, r)
			if r.StatusCode != 200 {
				t.Fatalf("connection %d: %d", i, r.StatusCode)
			}
			buf := make([]byte, 13)
			if _, e := io.ReadFull(r.Body, buf); e != nil || string(buf) != ": heartbeat\n\n" {
				t.Fatalf("buffered SSE: %q %v", buf, e)
			}
		}
		if status, _, _ := get(t, base, "/api/notification/v1/barrage/stream"); status != 429 {
			t.Fatalf("connection limit: %d", status)
		}
		cancels[0]()
		responses[0].Body.Close()
		time.Sleep(150 * time.Millisecond)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, "GET", base+"/api/notification/v1/barrage/stream", nil)
		r, e := streamClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer r.Body.Close()
		if r.StatusCode != 200 {
			t.Fatalf("released slot unavailable: %d", r.StatusCode)
		}
	})
}

const upstreamSource = `package main
import ("encoding/json"; "fmt"; "net/http"; "strings"; "time")
func main(){http.HandleFunc("/",func(w http.ResponseWriter,r *http.Request){
 if strings.HasSuffix(r.URL.Path,"/stream") {w.Header().Set("Content-Type","text/event-stream");fmt.Fprint(w,": heartbeat\n\n");w.(http.Flusher).Flush();tick:=time.NewTicker(15*time.Second);defer tick.Stop();for {select {case <-r.Context().Done():return;case <-tick.C:fmt.Fprint(w,": heartbeat\n\n");w.(http.Flusher).Flush()}}}
 if strings.HasSuffix(r.URL.Path,"/upstream429"){w.Header().Set("Retry-After","17");w.WriteHeader(429);fmt.Fprint(w,"{\"error\":\"LOGIN_LIMITED\"}");return}
 json.NewEncoder(w).Encode(map[string]string{"xff":r.Header.Get("X-Forwarded-For"),"real":r.Header.Get("X-Real-IP")})
 });if err:=http.ListenAndServe(":8080",nil);err!=nil {panic(err)}}
`
