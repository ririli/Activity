package domain

import (
	"math"
	"testing"
)

func TestWalletRejectsInvalidMutationWithoutChangingState(t *testing.T) {
	for _, c := range []struct {
		asset string
		delta int64
	}{{Gold, -101}, {Gold, math.MinInt64}, {Petal, math.MaxInt64}, {"INVALID", 1}} {
		w := Wallet{Gold: 100, Petals: 1}
		before := w
		if _, err := w.Apply(c.asset, c.delta); err == nil || w != before {
			t.Fatalf("accepted %+v: %+v", c, w)
		}
	}
}
func TestRewardChoiceIsOneTime(t *testing.T) {
	g := RewardGrant{Status: "PENDING_CHOICE", Candidates: []Prize{{ID: 3, Name: "ring", Quantity: 1}}}
	if _, changed, err := g.Select("3"); err != nil || !changed {
		t.Fatal(err)
	}
	if _, _, err := g.Select("4"); err == nil {
		t.Fatal("accepted outside candidate set")
	}
	g.Status = "SELECTED"
	g.SelectedID = "3"
	g.SelectedName = "ring"
	g.Quantity = 1
	if p, changed, err := g.Select("3"); err != nil || changed || p.Name != "ring" {
		t.Fatal("same choice did not replay")
	}
	if _, _, err := g.Select("4"); err == nil {
		t.Fatal("allowed second selection")
	}
}
func TestProductQuote(t *testing.T) {
	p, err := Quote("52", 0)
	if err != nil || !p.Limited || p.Cost != 3000 || p.Petals != 52 {
		t.Fatalf("%+v %v", p, err)
	}
	if _, err = Quote("top-up", math.MaxInt64); err == nil {
		t.Fatal("overflow accepted")
	}
}
