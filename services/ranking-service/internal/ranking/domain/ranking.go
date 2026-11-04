package domain

import (
	"fmt"
	"math"
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
	User     string `json:"user_id"`
	Edition  string `json:"edition_id"`
	Draw     string `json:"draw_id"`
	Seq      string `json:"draw_seq"`
	Consumed string `json:"consumed_petals"`
	Occurred string `json:"occurred_at_sec"`
	Total    string `json:"total_score"`
	Reached  string `json:"score_reached_at_sec"`
	Created  string `json:"participation_created_at_sec"`
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
