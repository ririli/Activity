package domain

import (
	"activity/pkg/hourly"
	"fmt"
	"math"
	"slices"
	"strconv"
)

// Packed uses exact integer arithmetic before conversion to a Redis double.
func Packed(score, achieved, minSec, endSec int64) (int64, error) {
	span := endSec - minSec
	if span < 0 || span >= math.MaxInt64-1 || score < 0 || achieved < minSec || achieved > endSec {
		return 0, fmt.Errorf("invalid ranking encoding")
	}
	base := span + 1
	offset := endSec - achieved
	if score > (1<<53-offset)/base {
		return 0, fmt.Errorf("ranking precision exhausted")
	}
	return score*base + offset, nil
}
func Member(id uint64) string { return fmt.Sprintf("%020d", uint64(math.MaxUint64)-id) }

type Fact struct {
	User     string           `json:"user_id"`
	Edition  string           `json:"edition_id"`
	Draw     string           `json:"draw_id"`
	Seq      string           `json:"draw_seq"`
	Consumed string           `json:"consumed_petals"`
	Occurred string           `json:"occurred_at_sec"`
	Total    string           `json:"total_score"`
	Reached  string           `json:"score_reached_at_sec"`
	Created  string           `json:"participation_created_at_sec"`
	Hourly   *hourly.Position `json:"hourly,omitempty"`
}

type Entry struct {
	User     string `json:"user_id"`
	Score    int64  `json:"score"`
	Reached  int64  `json:"score_reached_at_sec"`
	Rank     int64  `json:"rank"`
	Nickname string `json:"nickname"`
}

// Board is the score/rank projection before display names are enriched by the
// application layer.
type Board struct {
	Top    []Entry `json:"top"`
	Me     *Entry  `json:"me"`
	Source string  `json:"source"`
}

// Top merges disjoint shard results and/or earlier winners with current changes.
func Top(entries []Entry, limit int) []Entry {
	byUser := make(map[string]Entry, len(entries))
	for _, entry := range entries {
		old, found := byUser[entry.User]
		if !found || entry.Score > old.Score || entry.Score == old.Score && entry.Reached < old.Reached {
			byUser[entry.User] = entry
		}
	}
	ordered := make([]Entry, 0, len(byUser))
	for _, entry := range byUser {
		ordered = append(ordered, entry)
	}
	slices.SortFunc(ordered, func(a, b Entry) int {
		if a.Score != b.Score {
			if a.Score > b.Score {
				return -1
			}
			return 1
		}
		if a.Reached != b.Reached {
			if a.Reached < b.Reached {
				return -1
			}
			return 1
		}
		// All IDs are canonical decimal strings. Length then lexical order is numeric order.
		if len(a.User) != len(b.User) {
			return len(a.User) - len(b.User)
		}
		if a.User < b.User {
			return -1
		}
		if a.User > b.User {
			return 1
		}
		return 0
	})
	ordered = ordered[:min(limit, len(ordered))]
	for i := range ordered {
		ordered[i].Rank = int64(i + 1)
	}
	return ordered
}

func Unpack(member string, packed, minSec, endSec int64) (Entry, error) {
	id, err := strconv.ParseUint(member, 10, 64)
	if err != nil || len(member) != 20 || id == math.MaxUint64 || packed < 0 || packed > 1<<53 || minSec < 0 || endSec < minSec || endSec > 1<<53-1 {
		return Entry{}, fmt.Errorf("invalid ranking member")
	}
	base := endSec - minSec + 1
	return Entry{User: strconv.FormatUint(math.MaxUint64-id, 10), Score: packed / base, Reached: endSec - packed%base}, nil
}
