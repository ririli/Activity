package platform

import (
	"activity/pkg/ratelimit"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAdmissionResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	Respond(c, nil, fmt.Errorf("wrapped: %w", &ratelimit.Rejection{Status: 429, Code: "RATE_LIMITED", After: 1001 * time.Millisecond}))
	if w.Code != 429 || w.Header().Get("Retry-After") != "2" || w.Body.String() != `{"error":"RATE_LIMITED"}` {
		t.Fatalf("%d %v %s", w.Code, w.Header(), w.Body.String())
	}
}
