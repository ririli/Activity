package domain

import (
	"math"
	"testing"
)

func TestRankingOrder(t *testing.T) {
	a, e := Packed(100, 10, 0, 1000)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := Packed(100, 11, 0, 1000)
	c, _ := Packed(101, 999, 0, 1000)
	if !(a > b && c > a) {
		t.Fatal("score/time order incorrect")
	}
	if Member(1) <= Member(2) {
		t.Fatal("reverse member order incorrect")
	}
	if _, e = Packed(math.MaxInt64, 10, 0, 1000); e == nil {
		t.Fatal("precision overflow accepted")
	}
}
