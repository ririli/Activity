package domain

import (
	"math"
	"testing"
)

func TestAdvanceProgressGuaranteesAndBox(t *testing.T) {
	rules := make([]FlowerRule, MaxFlowers)
	for i := range rules {
		rules[i].Guarantee = int64(i+1) * 1000
	}
	after, boxes, err := AdvanceProgress(Progress{Flowers: 2}, 100, 999999, rules)
	if err != nil || after.Flowers != 6 || after.GoldValue != 6000 || boxes != 1 {
		t.Fatalf("got %+v, boxes %d, err %v", after, boxes, err)
	}
}

func TestAdvanceProgressOneProbabilityCheck(t *testing.T) {
	rules := make([]FlowerRule, MaxFlowers)
	for i := range rules {
		rules[i] = FlowerRule{Probability: ProbabilityScale, Guarantee: 100000}
	}
	after, _, err := AdvanceProgress(Progress{}, 1, 0, rules)
	if err != nil || after.Flowers != 1 {
		t.Fatalf("got %+v, %v", after, err)
	}
}

func TestAdvanceProgressOverflowDoesNotChangeInput(t *testing.T) {
	rules := make([]FlowerRule, MaxFlowers)
	before := Progress{GoldValue: math.MaxInt64}
	after, _, err := AdvanceProgress(before, 1, 0, rules)
	if err == nil || after != before {
		t.Fatalf("got %+v, %v", after, err)
	}
}
