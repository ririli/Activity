package domain

import (
	lottery "activity/services/activity-service/internal/lottery/domain"
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
	Cost    int64                `json:"cost"`
	Prizes  []Prize              `json:"prizes"`
	Flowers []lottery.FlowerRule `json:"flowers"`
}
type Rules struct {
	Pools  map[string]Pool `json:"pools"`
	Stages map[int]Prize   `json:"stages"`
	Chests []Prize         `json:"chests"`
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
func Simulate(p Pool, before lottery.Progress, count int, rng func(int64) (int64, error)) (Plan, error) {
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
	dist, e := lottery.NewWeightedPool(weights)
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
		sample, e := rng(lottery.ProbabilityScale)
		if e != nil {
			return out, e
		}
		old := lottery.Progress{Flowers: out.Flowers, GoldValue: out.GoldValue}
		next, _, e := lottery.AdvanceProgress(old, p.Cost, sample, p.Flowers)
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
func DefaultRules() Rules {
	choice := Prize{Code: "TRUE_LOVE_CHOICE", Name: "真爱无敌戒指选择权", Quantity: 1, Choices: []Prize{{Code: "1207751", ID: 1207751, Name: "真爱无敌戒指经典款", Quantity: 1}, {Code: "1207752", ID: 1207752, Name: "真爱无敌戒指铭文款", Quantity: 1}, {Code: "1207753", ID: 1207753, Name: "爱你万年戒指铭文款", Quantity: 1}}}
	gold := func(q, w int64) Prize { return Prize{Code: "1001", ID: 1001, Name: "金币", Quantity: q, Weight: w} }
	petal := func(q, w int64) Prize { return Prize{Code: "1002", ID: 1002, Name: "花瓣", Quantity: q, Weight: w} }
	ring := func(id uint64, name string, w int64) Prize {
		return Prize{Code: fmt.Sprint(id), ID: id, Name: name, Quantity: 1, Weight: w}
	}
	a := ring(1207244, "烟花之恋戒指", 100)
	b := ring(1203743, "闪闪心蝶戒指", 70)
	c := ring(1205251, "月亮游记戒指", 25)
	d := ring(1205470, "爱意翩跹戒指", 70)
	dayChoice := choice
	dayChoice.Weight = 5
	nightChoice := choice
	nightChoice.Weight = 30
	day := []Prize{gold(100, 350), gold(300, 250), petal(1, 200), a, b, c, dayChoice}
	a.Weight = 200
	b.Weight = 150
	c.Weight = 100
	night := []Prize{gold(1000, 250), petal(10, 200), a, b, c, d, nightChoice}
	threshold := []int64{300, 900, 3500, 10000, 30000, 53198, 56000, 150000, 250000, 350000, 500000, 710399, 800000, 980000, 1170000, 1367702, 1672100, 2082499}
	dayProb := []int64{1000000, 25000, 5770, 2310, 750, 650, 5350, 160, 150, 150, 100, 70, 170, 80, 80, 80, 50, 40}
	nightProb := []int64{1000000, 1000000, 1000000, 46150, 15000, 12930, 107060, 3190, 3000, 3000, 2000, 1430, 3350, 1670, 1580, 1520, 990, 730}
	dr, nr := make([]lottery.FlowerRule, 18), make([]lottery.FlowerRule, 18)
	for i := range dr {
		dr[i] = lottery.FlowerRule{Probability: dayProb[i], Guarantee: threshold[i]}
		nr[i] = lottery.FlowerRule{Probability: nightProb[i], Guarantee: threshold[i]}
	}
	return Rules{Pools: map[string]Pool{"DAY": {5, day, dr}, "NIGHT": {100, night, nr}}, Stages: map[int]Prize{3: gold(500, 0), 5: petal(10, 0), 8: a, 11: gold(2000, 0), 15: c, 17: choice}, Chests: []Prize{c, d, choice}}
}
