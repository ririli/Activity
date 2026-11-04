package domain

import (
	"math"
	"testing"
)

func TestWeightedPoolIntervals(t *testing.T) {
	p, err := NewWeightedPool([]int64{2, 0, 3})
	if err != nil {
		t.Fatal(err)
	}
	for sample, want := range []int{0, 0, 2, 2, 2} {
		got, err := p.Select(int64(sample))
		if err != nil || got != want {
			t.Fatalf("sample %d: got %d, %v; want %d", sample, got, err, want)
		}
	}
	for _, sample := range []int64{-1, 5} {
		if _, err := p.Select(sample); err == nil {
			t.Fatalf("accepted sample %d", sample)
		}
	}
}

func TestWeightedPoolRejectsInvalidWeights(t *testing.T) {
	for _, weights := range [][]int64{nil, {0, 0}, {1, -1}, {math.MaxInt64, 1}} {
		if _, err := NewWeightedPool(weights); err == nil {
			t.Fatalf("accepted %v", weights)
		}
	}
}
