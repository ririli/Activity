package platform

import "testing"

func TestUserIDsRotateAcrossMilliseconds(t *testing.T) {
	var sequence idSequence
	seen := make(map[uint64]bool)
	var allocated uint64
	for ms := int64(100); ms < 124; ms++ {
		// Alternate a quiet millisecond and bursts with ordinary IDs interleaved.
		for range 1 + ms%2*2 {
			ordinary := sequence.allocate(ms, 9, false)
			user := sequence.allocate(ms, 9, true)
			if user%4 != allocated%4 || user>>22 != uint64(ms) || user>>12&1023 != 9 {
				t.Fatalf("unexpected user ID %d at allocation %d", user, allocated)
			}
			if user <= ordinary || seen[user] || seen[ordinary] {
				t.Fatalf("IDs must be unique and increasing: %d, %d", ordinary, user)
			}
			seen[ordinary], seen[user] = true, true
			allocated++
		}
	}
}

func TestUserIDSequenceDoesNotOverflowNodeBits(t *testing.T) {
	sequence := idSequence{lastMS: 100, next: 4095, userSlot: 0}
	defer func() {
		if got := recover(); got != "ID rate exhausted" {
			t.Fatalf("expected sequence exhaustion, got %v", got)
		}
		if user := sequence.allocate(101, 9, true); user%4 != 0 || user>>12&1023 != 9 {
			t.Fatalf("failed allocation must preserve the next user slot: %d", user)
		}
	}()
	sequence.allocate(100, 9, true)
}
