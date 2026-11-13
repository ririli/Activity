package domain

import (
	"testing"
	"time"
)

func TestRoundCompletionAndStageClaim(t *testing.T) {
	p := Progress{Round: 1, Flowers: 18, Completed: 3, Mask: 63}
	if err := p.CheckNext(1, 0); err != nil {
		t.Fatal(err)
	}
	if err := p.CheckNext(1, 1); err == nil {
		t.Fatal("pending choice ignored")
	}
	p.Completed = 2
	if err := p.CheckNext(1, 0); err == nil {
		t.Fatal("unconfirmed chest ignored")
	}
	p = Progress{Flowers: 3}
	if err := p.Claim(3); err != nil || p.Mask != 1 {
		t.Fatalf("%+v %v", p, err)
	}
	before := p
	if err := p.Claim(5); err == nil || p != before {
		t.Fatal("unreached stage accepted")
	}
}
func TestChestConfirmation(t *testing.T) {
	c := Chest{Status: "OPENED", RewardStatus: "OBTAINED", Round: 1}
	if err := c.CheckConfirm(1); err != nil {
		t.Fatal(err)
	}
	c.RewardStatus = "PENDING_CHOICE"
	if err := c.CheckConfirm(1); err == nil {
		t.Fatal("unresolved choice accepted")
	}
	c.Status = "CONFIRMED"
	if err := c.CheckConfirm(2); err != nil {
		t.Fatal("confirmed chest replay failed")
	}
}
func TestEditionAdmissionAtDeadline(t *testing.T) {
	now := time.Now()
	e := Edition{Status: "ACTIVE", Start: now.Add(-time.Hour), End: now, Now: now}
	if err := e.CheckActive(); err == nil {
		t.Fatal("accepted at exclusive end")
	}
	e.Now = now.Add(-time.Nanosecond)
	if err := e.CheckActive(); err != nil {
		t.Fatal(err)
	}
}
