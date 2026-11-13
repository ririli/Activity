// Package hourly defines the internal contracts for hourly leaderboard settlement.
package hourly

import "time"

const (
	Shards      = 4
	Lanes       = 16
	Winners     = 100
	Petals      = 1000
	BarrierType = "HourlySettlementBarrier.v1"
)

// China Standard Time is UTC+08:00 for all supported activity editions.
var location = time.FixedZone("Asia/Shanghai", 8*60*60)

func NextCutoff(at time.Time) time.Time {
	local := at.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), 0, 0, 0, location).Add(time.Hour).UTC()
}

func Shard(user uint64) int { return int(user % Shards) }
func Lane(user uint64) int  { return int((user / Shards) % Lanes) }

type Edition struct {
	ID    string    `json:"edition_id"`
	Start time.Time `json:"starts_at"`
	End   time.Time `json:"ends_at"`
}

type Window struct {
	Edition string `json:"edition_id"`
	Cutoff  int64  `json:"cutoff_sec"`
}

type Position struct {
	Cutoff int64 `json:"cutoff_sec"`
	Shard  int   `json:"hourly_shard"`
	Lane   int   `json:"hourly_lane"`
	Seq    int64 `json:"hourly_seq"`
}

type Watermark [Shards][Lanes]int64

type Replay struct {
	Window
	Shard   int   `json:"shard"`
	Lane    int   `json:"lane"`
	After   int64 `json:"after"`
	Through int64 `json:"through"`
}

type Positions struct {
	Window
	Shard int `json:"shard"`
}

type Barrier struct {
	Window
	Settlement string `json:"settlement_id"`
}

type Grant struct {
	Barrier
	User string `json:"user_id"`
	Rank int    `json:"rank"`
}

type Receipt struct {
	GrantID string `json:"grant_id"`
}
