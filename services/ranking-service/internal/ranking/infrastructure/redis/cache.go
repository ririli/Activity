package redis

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/domain"
	rdb "github.com/redis/go-redis/v9"
)

//go:embed guard.lua
var guardLua string

//go:embed update.lua
var updateLua string

//go:embed read.lua
var readLua string

var update = rdb.NewScript(guardLua + "\n" + updateLua)
var read = rdb.NewScript(guardLua + "\n" + readLua)
var ErrUnavailable = errors.New("ranking cache unavailable")

type Bounds struct {
	Min int64 `json:"min_sec"`
	End int64 `json:"end_sec"`
}
type Cache struct {
	Client     *rdb.Client
	LoadBounds func(context.Context, string) (Bounds, error)
}

// Generation is published by the repository only after the full rebuild.
type Generation struct {
	id   string
	keys []string
}

func prefix(edition string) string {
	return "ranking:" + platform.Env("APP_ENV", "local") + ":{" + edition + "}:"
}

func legacyKeys(p string) []string {
	return []string{p + "board", p + "users", p + "draws", p + "bounds", p + "disabled"}
}

func generationKeys(p string) []string {
	p += "v2:"
	return []string{p + "meta", p + "board", p + "users", p + "draws"}
}

func (c *Cache) Start(ctx context.Context, edition string) (*Generation, error) {
	b, err := c.LoadBounds(ctx, edition)
	if err != nil {
		return nil, err
	}
	if b.Min < 0 || b.End < b.Min || b.End > 1<<53-1 {
		return nil, ErrUnavailable
	}
	info, err := c.Client.Info(ctx, "server").Result()
	if err != nil {
		return nil, err
	}
	var run string
	for _, line := range strings.Split(info, "\r\n") {
		if value, ok := strings.CutPrefix(line, "run_id:"); ok {
			run = value
			break
		}
	}
	if run == "" {
		return nil, ErrUnavailable
	}
	p := prefix(edition)
	g := &Generation{id: platform.Random(), keys: generationKeys(p)}
	// Reuse the physical keys only after the old generation is unpublished.
	// The generation token prevents in-flight reads from seeing partial data.
	_, err = c.Client.TxPipelined(ctx, func(pipe rdb.Pipeliner) error {
		pipe.Unlink(ctx, append(legacyKeys(p), g.keys...)...)
		pipe.HSet(ctx, g.keys[0], "generation", g.id, "run_id", run, "min", b.Min, "end", b.End, "users", 0, "draws", 0)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := update.Load(ctx, c.Client).Err(); err != nil {
		return nil, err
	}
	return g, nil
}

func (c *Cache) Retire(ctx context.Context, edition string) error {
	p := prefix(edition)
	return c.Client.Unlink(ctx, append(legacyKeys(p), generationKeys(p)...)...).Err()
}

func updateArgs(g *Generation, f domain.Fact) ([]any, error) {
	u, err := platform.UID(f.User)
	if err != nil {
		return nil, err
	}
	delta, at := "0", f.Created
	if f.Draw != "" {
		delta, at = f.Consumed, f.Occurred
	}
	return []any{g.id, f.User, domain.Member(u), f.Draw, delta, at}, nil
}

func (c *Cache) Apply(ctx context.Context, g *Generation, f domain.Fact) error {
	args, err := updateArgs(g, f)
	if err != nil {
		return err
	}
	n, err := update.Run(ctx, c.Client, g.keys, args...).Int()
	if err == nil && n < 0 {
		return ErrUnavailable
	}
	return err
}

func (c *Cache) ApplyBatch(ctx context.Context, g *Generation, facts []domain.Fact) error {
	commands := make([]*rdb.Cmd, 0, len(facts))
	_, err := c.Client.Pipelined(ctx, func(pipe rdb.Pipeliner) error {
		for _, f := range facts {
			args, err := updateArgs(g, f)
			if err != nil {
				return err
			}
			commands = append(commands, update.EvalSha(ctx, pipe, g.keys, args...))
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, cmd := range commands {
		n, err := cmd.Int()
		if err != nil {
			return err
		}
		if n < 0 {
			return ErrUnavailable
		}
	}
	return nil
}

func (c *Cache) Board(ctx context.Context, g *Generation, user string) (domain.Board, error) {
	u, err := platform.UID(user)
	if err != nil {
		return domain.Board{}, err
	}
	result, err := read.Run(ctx, c.Client, g.keys, g.id, domain.Member(u)).StringSlice()
	if err != nil {
		return domain.Board{}, err
	}
	if len(result) < 2 {
		return domain.Board{}, ErrUnavailable
	}
	b := domain.Board{Top: make([]domain.Entry, 0, len(result)-2), Source: "redis"}
	if result[0] != "" {
		v, err := decodeEntry(result[0])
		if err != nil {
			return b, err
		}
		v.Rank, err = strconv.ParseInt(result[1], 10, 64)
		if err != nil {
			return b, err
		}
		b.Me = &v
	}
	for i, raw := range result[2:] {
		v, err := decodeEntry(raw)
		if err != nil {
			return b, err
		}
		v.Rank = int64(i + 1)
		b.Top = append(b.Top, v)
	}
	return b, nil
}

func decodeEntry(raw string) (domain.Entry, error) {
	var m struct {
		User    string `json:"user_id"`
		Score   int64  `json:"score,string"`
		Reached int64  `json:"reached,string"`
	}
	err := json.Unmarshal([]byte(raw), &m)
	return domain.Entry{User: m.User, Score: m.Score, Reached: m.Reached}, err
}
