package domain

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
)

type Prize struct {
	Code     string  `json:"code"`
	ID       uint64  `json:"item_id,string"`
	Name     string  `json:"name"`
	Quantity int64   `json:"quantity"`
	Weight   int64   `json:"weight"`
	Choices  []Prize `json:"choices,omitempty"`
}

type Pool struct {
	Cost    int64        `json:"cost"`
	Prizes  []Prize      `json:"prizes"`
	Flowers []FlowerRule `json:"flowers"`
}

type Step struct {
	Prize       Prize `json:"prize"`
	Before      int   `json:"flowers_before"`
	After       int   `json:"flowers_after"`
	Probability int   `json:"probability_lit"`
	GoldValue   int64 `json:"gold_value"`
}

type Plan struct {
	Steps     []Step `json:"steps"`
	Cost      int64  `json:"cost"`
	Refund    int64  `json:"refund"`
	Flowers   int    `json:"flowers"`
	GoldValue int64  `json:"gold_value"`
}

func Random(n int64) (int64, error) {
	if n <= 0 {
		return 0, errors.New("invalid random bound")
	}
	v, e := rand.Int(rand.Reader, big.NewInt(n))
	if e != nil {
		return 0, e
	}
	return v.Int64(), nil
}

func Simulate(p Pool, before Progress, count int, rng func(int64) (int64, error)) (Plan, error) {
	out := Plan{Flowers: before.Flowers, GoldValue: before.GoldValue}
	if p.Cost <= 0 || count <= 0 || count > 180 || p.Cost > math.MaxInt64/int64(count) || before.Flowers >= 18 {
		return out, errors.New("invalid plan")
	}
	weights := make([]int64, len(p.Prizes))
	var total int64
	for i, r := range p.Prizes {
		if r.Quantity <= 0 || r.Weight < 0 || r.Weight > math.MaxInt64-total {
			return out, errors.New("invalid prize")
		}
		weights[i] = r.Weight
		total += r.Weight
	}
	dist, e := NewWeightedPool(weights)
	if e != nil {
		return out, e
	}
	for i := 0; i < count && out.Flowers < 18; i++ {
		r, e := rng(total)
		if e != nil {
			return out, e
		}
		index, e := dist.Select(r)
		if e != nil {
			return out, e
		}
		sample, e := rng(ProbabilityScale)
		if e != nil {
			return out, e
		}
		old := Progress{Flowers: out.Flowers, GoldValue: out.GoldValue}
		next, _, e := AdvanceProgress(old, p.Cost, sample, p.Flowers)
		if e != nil {
			return out, e
		}
		prob := 0
		if sample < p.Flowers[old.Flowers].Probability {
			prob = 1
		}
		out.Steps = append(out.Steps, Step{p.Prizes[index], old.Flowers, next.Flowers, prob, next.GoldValue})
		out.Flowers = next.Flowers
		out.GoldValue = next.GoldValue
	}
	out.Cost = int64(len(out.Steps)) * p.Cost
	out.Refund = int64(count)*p.Cost - out.Cost
	return out, nil
}

func (p Prize) String() string { return fmt.Sprintf("%s(%d)", p.Name, p.ID) }
