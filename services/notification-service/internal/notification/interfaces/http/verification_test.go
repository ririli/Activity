package http

import (
	"activity/pkg/platform"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestVerificationDistinguishesSessionFailureFromDependencyFailure(t *testing.T) {
	for _, tt := range []struct {
		err   error
		event string
	}{
		{platform.Fail(401, "INVALID_SESSION"), "auth_required"},
		{platform.Fail(403, "ACCOUNT_DISABLED"), "auth_required"},
		{platform.Fail(503, "DEPENDENCY_UNAVAILABLE"), "unavailable"},
		{context.DeadlineExceeded, "unavailable"},
		{errors.New("connection refused"), "unavailable"},
	} {
		if got := verificationFailure(tt.err); !strings.HasPrefix(got, "event: "+tt.event+"\n") {
			t.Fatalf("%v: %q", tt.err, got)
		}
	}
}
