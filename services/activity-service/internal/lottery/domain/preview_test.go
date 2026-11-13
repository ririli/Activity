package domain

import "testing"

func TestPreviewStartingPointAndReplay(t *testing.T) {
	p := Preview{Status: "PENDING", Round: 1, Version: 3}
	if err := p.CheckConfirm(1, 3, 0); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][3]int64{{2, 3, 0}, {1, 4, 0}, {1, 3, 1}} {
		if err := p.CheckConfirm(int(args[0]), args[1], args[2]); err == nil {
			t.Fatal("stale preview accepted")
		}
	}
	p.Status = "CONFIRMED"
	if err := p.CheckConfirm(2, 9, 10); err != nil {
		t.Fatal("confirmed result not recoverable")
	}
}
