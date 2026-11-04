package redis

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/domain"
	rdb "github.com/redis/go-redis/v9"
)

//go:embed update.lua
var updateLua string

//go:embed read.lua
var readLua string
var update = rdb.NewScript(updateLua)
var read = rdb.NewScript(readLua)
var ErrUnavailable = errors.New("ranking cache unavailable")

type Bounds struct {
	Min int64 `json:"min_sec"`
	End int64 `json:"end_sec"`
}
type Cache struct {
	Client     *rdb.Client
	LoadBounds func(context.Context, string) (Bounds, error)
	bounds     sync.Map
}

func (c *Cache) keys(edition string) []string {
	prefix := "ranking:" + platform.Env("APP_ENV", "local") + ":{" + edition + "}:"
	return []string{prefix + "board", prefix + "users", prefix + "draws", prefix + "bounds", prefix + "disabled"}
}
func (c *Cache) Disable(ctx context.Context, edition string) error {
	return c.Client.Set(ctx, c.keys(edition)[4], "missing-history", 0).Err()
}
func (c *Cache) HasUser(ctx context.Context, edition, user string) (bool, error) {
	return c.Client.HExists(ctx, c.keys(edition)[1], user).Result()
}
func (c *Cache) Apply(ctx context.Context, ev platform.Event) error {
	if ev.Type != "FormalDrawCommitted.v1" && ev.Type != "ParticipationInitialized.v1" {
		return platform.Fail(409, "INVALID_CACHE_EVENT")
	}
	var f domain.Fact
	if e := json.Unmarshal(ev.Payload, &f); e != nil {
		return platform.Fail(409, "INVALID_CACHE_EVENT")
	}
	user, e := platform.UID(f.User)
	if e != nil {
		return e
	}
	if fmt.Sprint(user) != f.User {
		return platform.Fail(409, "NON_CANONICAL_USER_ID")
	}
	if _, e = platform.UID(f.Edition); e != nil {
		return e
	}
	atText := f.Created
	deltaText := "0"
	draw := ""
	fingerprint := ""
	if ev.Type == "FormalDrawCommitted.v1" {
		if _, e = platform.UID(f.Draw); e != nil {
			return e
		}
		draw = f.Draw
		atText = f.Occurred
		deltaText = f.Consumed
		delta, e := strconv.ParseInt(deltaText, 10, 64)
		if e != nil || delta <= 0 {
			return platform.Fail(409, "INVALID_CACHE_DELTA")
		}
		if delta > 1<<53-1 {
			return c.Disable(ctx, f.Edition)
		}
		h := sha256.Sum256(platform.JSON([]string{f.User, f.Edition, draw, deltaText, atText, f.Seq}))
		fingerprint = hex.EncodeToString(h[:])
	}
	at, e := strconv.ParseInt(atText, 10, 64)
	if e != nil || at < 0 || at > 1<<53-1 {
		return platform.Fail(409, "INVALID_CACHE_TIME")
	}
	var b Bounds
	if v, ok := c.bounds.Load(f.Edition); ok {
		b = v.(Bounds)
	} else {
		b, e = c.LoadBounds(ctx, f.Edition)
		if e != nil {
			return e
		}
		if b.Min < 0 || b.End < b.Min || b.End > 1<<53-1 {
			return platform.Fail(409, "INVALID_CACHE_BOUNDS")
		}
		c.bounds.Store(f.Edition, b)
	}
	n, e := update.Run(ctx, c.Client, c.keys(f.Edition), f.User, domain.Member(user), draw, fingerprint, deltaText, atText, fmt.Sprint(b.Min), fmt.Sprint(b.End)).Int()
	if e != nil {
		return e
	}
	if n == -1 {
		return platform.Fail(409, "DRAW_PAYLOAD_CONFLICT")
	}
	if n == -3 {
		return platform.Fail(409, "INVALID_CACHE_TIME")
	}
	return nil
}

type Board struct {
	Top   []domain.Entry
	Me    *domain.Entry
	Count int64
}

func (c *Cache) Board(ctx context.Context, edition, user string) (Board, error) {
	u, e := platform.UID(user)
	if e != nil {
		return Board{}, e
	}
	result, e := read.Run(ctx, c.Client, c.keys(edition), user, domain.Member(u)).Slice()
	if e != nil {
		return Board{}, e
	}
	if len(result) == 0 {
		return Board{}, ErrUnavailable
	}
	var b Board
	b.Top = []domain.Entry{}
	if e = json.Unmarshal([]byte(result[0].(string)), &b.Count); e != nil {
		return b, e
	}
	decode := func(raw string) (domain.Entry, error) {
		var m struct {
			User    string `json:"user_id"`
			Score   string `json:"score"`
			Reached string `json:"reached"`
		}
		e := json.Unmarshal([]byte(raw), &m)
		if e != nil {
			return domain.Entry{}, e
		}
		score, e := strconv.ParseInt(m.Score, 10, 64)
		if e != nil {
			return domain.Entry{}, e
		}
		reached, e := strconv.ParseInt(m.Reached, 10, 64)
		return domain.Entry{User: m.User, Score: score, Reached: reached}, e
	}
	if raw, ok := result[1].(string); ok && raw != "" {
		v, e := decode(raw)
		if e != nil {
			return b, e
		}
		rank, e := strconv.ParseInt(result[2].(string), 10, 64)
		if e != nil {
			return b, e
		}
		v.Rank = rank
		b.Me = &v
	}
	for i := 3; i < len(result); i++ {
		v, e := decode(result[i].(string))
		if e != nil {
			return b, e
		}
		v.Rank = int64(i - 2)
		b.Top = append(b.Top, v)
	}
	return b, nil
}
