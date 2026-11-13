package domain

import (
	"activity/services/activity-service/internal/fault"
	"math"
	"time"
)

type Edition struct {
	Status          string
	Start, End, Now time.Time
	RuleVersion     int
	HourlyRanking   bool
}

func (e Edition) CheckActive() error {
	if e.Status != "ACTIVE" || e.Now.Before(e.Start) || !e.Now.Before(e.End) {
		return fault.New("ACTIVITY_NOT_ACTIVE")
	}
	return nil
}

type Progress struct {
	Round, Version, Flowers, Unlocked, Completed, Mask int
	Value, Formal, Seq, Progress, Score, Reached       int64
}

func (p Progress) CheckNext(from, pending int) error {
	if from != p.Round || p.Flowers != 18 || p.Completed != 3 || p.Mask != 63 {
		return fault.New("ROUND_NOT_COMPLETE")
	}
	if pending != 0 {
		return fault.New("STAGE_CHOICE_PENDING")
	}
	return nil
}
func (p *Progress) Claim(stage int) error {
	bit := 0
	for i, v := range []int{3, 5, 8, 11, 15, 17} {
		if v == stage {
			bit = 1 << i
		}
	}
	if bit == 0 {
		return fault.New("INVALID_STAGE")
	}
	if p.Flowers < stage {
		return fault.New("STAGE_NOT_REACHED")
	}
	p.Mask |= bit
	return nil
}
func (p *Progress) ApplyDraw(flowers int, value, cost, now int64) error {
	if flowers < p.Flowers || flowers > 18 || value < p.Value || cost <= 0 || cost > math.MaxInt64-p.Score || p.Seq == math.MaxInt64 || p.Formal == math.MaxInt64 || p.Progress == math.MaxInt64 {
		return fault.New("INVALID_DRAW_PROGRESS")
	}
	p.Flowers = flowers
	p.Value = value
	p.Unlocked = flowers / 6
	p.Score += cost
	p.Seq++
	p.Formal++
	p.Progress++
	p.Reached = max(now, p.Reached)
	return nil
}

type Chest struct {
	Status, RewardStatus string
	Round                int
}

func (c Chest) CheckConfirm(round int) error {
	if c.Status == "CONFIRMED" {
		return nil
	}
	if c.Round != round || c.RewardStatus == "PENDING_CHOICE" {
		return fault.New("CHEST_NOT_READY")
	}
	return nil
}

func (e Edition) CheckEnded() error {
	if e.Now.Before(e.End) {
		return fault.New("ACTIVITY_NOT_ENDED")
	}
	return nil
}
