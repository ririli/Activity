package application

import "testing"

func TestSettlementCursorMustBeEntireUnsignedNumber(t *testing.T) {
	for _, invalid := range []string{"12junk", "12 34", "-1", "+1", " 1", "9223372036854775808", ""} {
		if _, err := parseCursor(invalid); err == nil {
			t.Fatalf("accepted cursor %q", invalid)
		}
	}
	for _, valid := range []string{"0", "123", "9223372036854775807"} {
		if _, err := parseCursor(valid); err != nil {
			t.Fatalf("rejected %q: %v", valid, err)
		}
	}
}
