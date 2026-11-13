package redis

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"

	"activity/pkg/hourly"
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
	Clients    [hourly.Shards]*rdb.Client
	LoadBounds func(context.Context, string) (Bounds, error)
}

// Generation is published by the repository only after the full rebuild.
type Generation struct {
	id     string
	shards [][]string
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

func (c *Cache) clients() []*rdb.Client {
	if c.Clients[0] != nil {
		return c.Clients[:]
	}
	return []*rdb.Client{c.Client}
}

func livePrefix(edition string, shard, count int) string {
	if count == 1 {
		return prefix(edition)
	}
	return prefix(edition + ":" + strconv.Itoa(shard))
}

func (c *Cache) Start(ctx context.Context, edition string) (*Generation, error) {
	b, err := c.LoadBounds(ctx, edition)
	if err != nil {
		return nil, err
	}
	if b.Min < 0 || b.End < b.Min || b.End > 1<<53-1 {
		return nil, ErrUnavailable
	}
	clients := c.clients()
	if len(clients) > 1 && c.Client != nil {
		p := prefix(edition)
		// Old-format keys are unused by the new generation.
		_ = c.Client.Unlink(ctx, append(legacyKeys(p), generationKeys(p)...)...).Err()
	}
	g := &Generation{id: platform.Random(), shards: make([][]string, len(clients))}
	for shard, client := range clients {
		p := livePrefix(edition, shard, len(clients))
		keys := generationKeys(p)
		if err := startShard(ctx, client, p, keys, g.id, b); err != nil {
			return nil, err
		}
		g.shards[shard] = keys
	}
	return g, nil
}

func startShard(ctx context.Context, client *rdb.Client, p string, keys []string, generation string, b Bounds) error {
	info, err := client.Info(ctx, "server").Result()
	if err != nil {
		return err
	}
	var run string
	for _, line := range strings.Split(info, "\r\n") {
		if value, ok := strings.CutPrefix(line, "run_id:"); ok {
			run = value
			break
		}
	}
	if run == "" {
		return ErrUnavailable
	}
	// Reuse the physical keys only after the old generation is unpublished.
	// The generation token prevents in-flight reads from seeing partial data.
	_, err = client.TxPipelined(ctx, func(pipe rdb.Pipeliner) error {
		pipe.Unlink(ctx, append(legacyKeys(p), keys...)...)
		pipe.HSet(ctx, keys[0], "generation", generation, "run_id", run, "min", b.Min, "end", b.End, "users", 0, "draws", 0)
		return nil
	})
	if err != nil {
		return err
	}
	return update.Load(ctx, client).Err()
}

func (c *Cache) Retire(ctx context.Context, edition string) error {
	clients := c.clients()
	var failure error
	for shard, client := range clients {
		p := livePrefix(edition, shard, len(clients))
		failure = errors.Join(failure, client.Unlink(ctx, append(legacyKeys(p), generationKeys(p)...)...).Err())
	}
	if len(clients) > 1 && c.Client != nil {
		p := prefix(edition)
		failure = errors.Join(failure, c.Client.Unlink(ctx, append(legacyKeys(p), generationKeys(p)...)...).Err())
	}
	return failure
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
	user, _ := platform.UID(f.User)
	shard := int(user % uint64(len(g.shards)))
	n, err := update.Run(ctx, c.clients()[shard], g.shards[shard], args...).Int()
	if err == nil && n < 0 {
		return ErrUnavailable
	}
	return err
}

func (c *Cache) ApplyBatch(ctx context.Context, g *Generation, facts []domain.Fact) error {
	byShard := make([][]domain.Fact, len(g.shards))
	for _, f := range facts {
		user, err := platform.UID(f.User)
		if err != nil {
			return err
		}
		shard := int(user % uint64(len(g.shards)))
		byShard[shard] = append(byShard[shard], f)
	}
	for shard, batch := range byShard {
		if len(batch) > 0 {
			if err := applyBatch(ctx, c.clients()[shard], g, shard, batch); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyBatch(ctx context.Context, client *rdb.Client, g *Generation, shard int, facts []domain.Fact) error {
	commands := make([]*rdb.Cmd, 0, len(facts))
	_, err := client.Pipelined(ctx, func(pipe rdb.Pipeliner) error {
		for _, f := range facts {
			args, err := updateArgs(g, f)
			if err != nil {
				return err
			}
			commands = append(commands, update.EvalSha(ctx, pipe, g.shards[shard], args...))
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
	clients := c.clients()
	member := domain.Member(u)
	home := int(u % uint64(len(clients)))
	result, err := read.Run(ctx, clients[home], g.shards[home], g.id, member).StringSlice()
	if err != nil {
		return domain.Board{}, err
	}
	if len(result) < 3 {
		return domain.Board{}, ErrUnavailable
	}
	b := domain.Board{Source: "redis"}
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
	entries, err := decodeEntries(result[3:])
	if err != nil {
		return b, err
	}
	type shardResult struct {
		count   int64
		entries []domain.Entry
		err     error
	}
	others := make([]shardResult, len(clients))
	var reads sync.WaitGroup
	for shard, client := range clients {
		if shard == home {
			continue
		}
		reads.Go(func() {
			out := &others[shard]
			values, err := read.Run(ctx, client, g.shards[shard], g.id, member, result[2]).StringSlice()
			if err == nil && len(values) < 3 {
				err = ErrUnavailable
			}
			if err == nil {
				out.count, err = strconv.ParseInt(values[1], 10, 64)
			}
			if err == nil {
				out.entries, err = decodeEntries(values[3:])
			}
			out.err = err
		})
	}
	reads.Wait()
	for _, out := range others {
		if out.err != nil {
			return b, out.err
		}
		if b.Me != nil {
			b.Me.Rank += out.count
		}
		entries = append(entries, out.entries...)
	}
	b.Top = domain.Top(entries, 20)
	return b, nil
}

func decodeEntries(rawEntries []string) ([]domain.Entry, error) {
	entries := make([]domain.Entry, 0, len(rawEntries))
	for _, raw := range rawEntries {
		v, err := decodeEntry(raw)
		if err != nil {
			return nil, err
		}
		entries = append(entries, v)
	}
	return entries, nil
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
