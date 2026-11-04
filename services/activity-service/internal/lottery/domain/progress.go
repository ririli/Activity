package domain

import (
	"errors"
	"math"
)

const MaxFlowers = 18
const ProbabilityScale int64 = 1_000_000

// FlowerRule describes the next flower's independent chance and guarantee.
type FlowerRule struct {
	Probability int64
	Guarantee   int64
}

type Progress struct {
	Flowers   int
	GoldValue int64
}

// AdvanceProgress applies one actual draw to a copy of the progress. sample is
// uniformly drawn in [0, ProbabilityScale); callers provide an independent sample.
func AdvanceProgress(before Progress, petalCost, sample int64, rules []FlowerRule) (Progress, int, error) {
	if len(rules) != MaxFlowers || before.Flowers < 0 || before.Flowers >= MaxFlowers || before.GoldValue < 0 || petalCost <= 0 || sample < 0 || sample >= ProbabilityScale {
		return before, 0, errors.New("invalid draw progress")
	}
	for i, rule := range rules {
		if rule.Probability < 0 || rule.Probability > ProbabilityScale || rule.Guarantee < 0 || (i > 0 && rule.Guarantee < rules[i-1].Guarantee) {
			return before, 0, errors.New("invalid flower rules")
		}
	}
	if petalCost > (math.MaxInt64-before.GoldValue)/60 {
		return before, 0, errors.New("gold value overflow")
	}
	after := before
	after.GoldValue += petalCost * 60
	if sample < rules[after.Flowers].Probability {
		after.Flowers++
	}
	for after.Flowers < MaxFlowers && after.GoldValue >= rules[after.Flowers].Guarantee {
		after.Flowers++
	}
	return after, after.Flowers/6 - before.Flowers/6, nil
}
