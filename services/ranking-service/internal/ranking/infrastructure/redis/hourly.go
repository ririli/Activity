package redis

import (
	"activity/pkg/hourly"
	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/domain"
	"context"
	_ "embed"
	"fmt"
	"math"
	"strconv"

	rdb "github.com/redis/go-redis/v9"
)

//go:embed hourly_window.lua
var windowLua string

var windowScript = rdb.NewScript(windowLua)

type HourlyCache struct {
	Clients    [hourly.Shards]*rdb.Client
	Namespace  string
	LoadBounds func(context.Context, string) (Bounds, error)
}

func (c HourlyCache) run(ctx context.Context, w hourly.Window, shard int, op string, args ...any) ([]string, error) {
	prefix := fmt.Sprintf("ranking:%s:{%s:%d}:hourly:%d:", c.Namespace, w.Edition, shard, w.Cutoff)
	keys := []string{prefix + "meta", prefix + "board"}
	return windowScript.Run(ctx, c.Clients[shard], keys, append([]any{op, hourly.Lanes}, args...)...).StringSlice()
}

func (c HourlyCache) Positions(ctx context.Context, w hourly.Window, shard int) ([hourly.Lanes]int64, error) {
	var positions [hourly.Lanes]int64
	result, err := c.run(ctx, w, shard, "state")
	if err != nil {
		return positions, err
	}
	if len(result) != 1+hourly.Lanes {
		return positions, ErrUnavailable
	}
	for i, raw := range result[1:] {
		positions[i], err = strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return positions, err
		}
	}
	return positions, nil
}

// Apply returns the contiguous position. A gap is recovered by replaying the
// committed source facts in order, using this same script.
func (c HourlyCache) Apply(ctx context.Context, w hourly.Window, shard, lane int, facts []domain.Fact) (int64, error) {
	bounds, err := c.LoadBounds(ctx, w.Edition)
	if err != nil {
		return 0, err
	}
	args := make([]any, 1, 1+len(facts)*3)
	args[0] = lane
	for _, f := range facts {
		user, err := platform.UID(f.User)
		if err != nil {
			return 0, err
		}
		p := f.Hourly
		if p == nil || f.Edition != w.Edition || p.Cutoff != w.Cutoff || p.Shard != shard || p.Lane != lane ||
			hourly.Shard(user) != shard || hourly.Lane(user) != lane || p.Seq <= 0 || p.Seq > 1<<53-1 {
			return 0, fmt.Errorf("invalid hourly ranking position")
		}
		score, reached := int64(0), f.Created
		if f.Draw != "" {
			score, err = strconv.ParseInt(f.Total, 10, 64)
			if err != nil || score <= 0 {
				return 0, fmt.Errorf("invalid hourly ranking score")
			}
			reached = f.Reached
		}
		at, err := strconv.ParseInt(reached, 10, 64)
		if err != nil || at >= w.Cutoff {
			return 0, fmt.Errorf("invalid hourly ranking time")
		}
		packed, err := domain.Packed(score, at, bounds.Min, bounds.End)
		if err != nil {
			return 0, err
		}
		args = append(args, p.Seq, domain.Member(user), packed)
	}
	result, err := c.run(ctx, w, shard, "apply", args...)
	if err != nil {
		return 0, err
	}
	if len(result) == 1 && result[0] == "retired" {
		return -1, nil
	}
	if len(result) != 2 {
		return 0, ErrUnavailable
	}
	return strconv.ParseInt(result[1], 10, 64)
}

func (c HourlyCache) Freeze(ctx context.Context, w hourly.Window, shard int, mark [hourly.Lanes]int64) ([]domain.Entry, error) {
	bounds, err := c.LoadBounds(ctx, w.Edition)
	if err != nil {
		return nil, err
	}
	args := make([]any, 0, hourly.Lanes+1)
	for _, seq := range mark {
		args = append(args, seq)
	}
	result, err := c.run(ctx, w, shard, "freeze", append(args, hourly.Winners)...)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 || result[0] != "frozen" || len(result)%2 != 1 {
		return nil, ErrUnavailable
	}
	entries := make([]domain.Entry, 0, (len(result)-1)/2)
	for i := 1; i < len(result); i += 2 {
		score, err := strconv.ParseFloat(result[i+1], 64)
		if err != nil || score < 0 || score > 1<<53 || math.Trunc(score) != score || math.IsNaN(score) {
			return nil, fmt.Errorf("invalid packed hourly score %q", result[i+1])
		}
		entry, err := domain.Unpack(result[i], int64(score), bounds.Min, bounds.End)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (c HourlyCache) Retire(ctx context.Context, w hourly.Window, shard int) error {
	_, err := c.run(ctx, w, shard, "retire")
	return err
}
