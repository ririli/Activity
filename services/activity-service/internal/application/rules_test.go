package application

import (
	lottery "activity/services/activity-service/internal/lottery/domain"
	"testing"
)

func TestDefaultNightFirstDraw(t *testing.T) {
	p, e := lottery.Simulate(DefaultRules().Pools["NIGHT"], lottery.Progress{}, 1, func(n int64) (int64, error) { return 0, nil })
	if e != nil {
		t.Fatal(e)
	}
	if p.Flowers != 3 || p.Cost != 100 || p.Steps[0].Probability != 1 {
		t.Fatalf("unexpected plan: %+v", p)
	}
}
func TestFullRoundStopsAndRefunds(t *testing.T) {
	p, e := lottery.Simulate(DefaultRules().Pools["DAY"], lottery.Progress{Flowers: 17, GoldValue: 2082200}, 30, func(n int64) (int64, error) { return n - 1, nil })
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Steps) != 1 || p.Cost != 5 || p.Refund != 145 || p.Flowers != 18 {
		t.Fatalf("unexpected plan: %+v", p)
	}
}
func TestNoAdditionalProbabilityAfterGuarantee(t *testing.T) {
	rules := DefaultRules()
	p := rules.Pools["NIGHT"]
	p.Flowers[3].Probability = 1000000
	plan, e := lottery.Simulate(p, lottery.Progress{}, 1, func(n int64) (int64, error) { return 0, nil })
	if e != nil {
		t.Fatal(e)
	}
	if plan.Flowers != 3 {
		t.Fatalf("extra probability applied: %d", plan.Flowers)
	}
}
