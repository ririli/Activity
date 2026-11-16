package platform

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	mysql "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

func Env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func Must(err error) {
	if err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}
func OpenDB() *sql.DB {
	db := openMySQL()
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	return db
}
func openMySQL() *sql.DB {
	cfg, err := mysql.ParseDSN(os.Getenv("MYSQL_DSN"))
	Must(err)
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.Params = map[string]string{"time_zone": "'+00:00'"}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	Must(err)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db
}
func Redis() *redis.Client {
	return redis.NewClient(&redis.Options{Addr: Env("REDIS_ADDR", "localhost:16379"), Password: os.Getenv("REDIS_PASSWORD"), MaxRetries: -1, ReadTimeout: time.Second, WriteTimeout: time.Second, DialTimeout: time.Second})
}

type Fault struct {
	Status int
	Code   string
}

func (e *Fault) Error() string           { return e.Code }
func Fail(status int, code string) error { return &Fault{status, code} }
func Respond(c *gin.Context, v any, err error) {
	if err != nil {
		if errors.Is(err, ErrIDUnavailable) {
			slog.Warn("ID issuance deferred", "error", err)
			c.JSON(503, gin.H{"error": "ID_UNAVAILABLE"})
			return
		}
		var admission interface {
			HTTPStatus() int
			ErrorCode() string
			RetryAfter() time.Duration
		}
		if errors.As(err, &admission) {
			if d := admission.RetryAfter(); d > 0 {
				c.Header("Retry-After", strconv.FormatInt(int64((d+time.Second-1)/time.Second), 10))
			}
			c.JSON(admission.HTTPStatus(), gin.H{"error": admission.ErrorCode()})
			return
		}
		var dbErr *mysql.MySQLError
		if errors.As(err, &dbErr) && (dbErr.Number == 3572 || dbErr.Number == 1213 || dbErr.Number == 1205 || dbErr.Number == 1062) {
			c.JSON(409, gin.H{"error": "STATE_CONFLICT"})
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, gin.H{"error": "NOT_FOUND"})
			return
		}
		var f *Fault
		if errors.As(err, &f) {
			c.JSON(f.Status, gin.H{"error": f.Code})
			return
		}
		slog.Error("request failed", "error", err)
		c.JSON(503, gin.H{"error": "DEPENDENCY_UNAVAILABLE"})
		return
	}
	c.JSON(200, v)
}
func Bind(c *gin.Context, v any) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		c.JSON(400, gin.H{"error": "INVALID_REQUEST"})
		return false
	}
	return true
}
func Router(db *sql.DB, r *redis.Client, extraReady ...func() error) *gin.Engine {
	g := gin.New()
	g.Use(gin.Recovery(), func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Next()
	})
	var trusted []string
	if raw := os.Getenv("TRUSTED_PROXIES"); raw != "" {
		trusted = strings.Split(raw, ",")
	}
	Must(g.SetTrustedProxies(trusted))
	g.GET("/health/live", func(c *gin.Context) { c.Status(200) })
	g.GET("/health/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		if db != nil {
			if err := db.PingContext(ctx); err != nil {
				c.Status(503)
				return
			}
		}
		if r != nil {
			if err := r.Ping(ctx).Err(); err != nil {
				c.Status(503)
				return
			}
		}
		for _, ready := range extraReady {
			if ready != nil && ready() != nil {
				c.Status(503)
				return
			}
		}
		c.Status(200)
	})
	return g
}
func Internal(c *gin.Context) {
	want := os.Getenv("INTERNAL_TOKEN")
	got := c.GetHeader("X-Internal-Token")
	if len(want) < 32 || subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
		c.AbortWithStatus(401)
	}
}

type Principal struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Nickname  string `json:"nickname"`
}

var client = &http.Client{Timeout: 2 * time.Second}

func Call(ctx context.Context, base, path string, in, out any) error {
	return call(ctx, client, base, path, in, out)
}

// CallTimeout is for bounded internal operations such as settlement that need a
// longer deadline than the default two-second authentication/profile calls.
func CallTimeout(ctx context.Context, base, path string, in, out any, timeout time.Duration) error {
	return call(ctx, &http.Client{Timeout: timeout}, base, path, in, out)
}

func call(ctx context.Context, client *http.Client, base, path string, in, out any) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(base, "/")+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", os.Getenv("INTERNAL_TOKEN"))
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Fail(resp.StatusCode, "INTERNAL_REQUEST_FAILED")
	}
	return json.NewDecoder(http.MaxBytesReader(nil, resp.Body, 1<<20)).Decode(out)
}
func Verify(ctx context.Context, token string) (Principal, error) {
	var p Principal
	if err := localAccess(token); err != nil {
		return p, err
	}
	err := Call(ctx, Env("IDENTITY_URL", "http://localhost:8081"), "/internal/identity/v1/introspect", map[string]string{"token": token}, &p)
	return p, err
}
func Auth(c *gin.Context) {
	raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if raw == "" || raw == c.GetHeader("Authorization") {
		c.AbortWithStatus(401)
		return
	}
	p, err := Verify(c.Request.Context(), raw)
	if err != nil {
		Respond(c, nil, err)
		c.Abort()
		return
	}
	c.Set("principal", p)
	c.Next()
}
func User(c *gin.Context) string { return c.MustGet("principal").(Principal).UserID }
func UID(s string) (uint64, error) {
	u, e := strconv.ParseUint(s, 10, 63)
	if e != nil || u == 0 {
		return 0, Fail(400, "INVALID_ID")
	}
	return u, nil
}
func Random() string { b := make([]byte, 16); MustRandom(b); return hex.EncodeToString(b) }
func MustRandom(b []byte) {
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
}

func Run(g *gin.Engine, worker func(context.Context)) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if worker != nil {
		go worker(ctx)
	}
	s := &http.Server{Addr: Env("HTTP_ADDR", ":8080"), Handler: g, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server failed", "error", err)
			stop()
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = s.Shutdown(shutdown)
}
func Shard(table string, user uint64) string { return fmt.Sprintf("%s_%d", table, user%4) }
func JSON(v any) []byte {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return b
}
