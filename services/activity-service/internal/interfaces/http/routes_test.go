package http

import (
	"activity/services/activity-service/internal/fault"
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

type useCaseStub struct {
	UseCases
	action, edition string
}

func (s *useCaseStub) Internal(_ context.Context, action, edition, user, after string) (any, error) {
	s.action = action
	s.edition = edition
	return map[string]string{"status": "ok"}, nil
}
func TestRoutesAndInternalDispatchRemainCompatible(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &useCaseStub{}
	g := gin.New()
	Routes(g, s)
	routes := map[string]bool{}
	for _, r := range g.Routes() {
		routes[r.Method+" "+r.Path] = true
	}
	for _, action := range []string{"join", "draw", "purchase", "stage", "chest-open", "chest-confirm", "select", "next-round", "preview", "preview-confirm", "preview-abandon", "tasks/claim"} {
		if !routes["POST /api/activity/v1/"+action] {
			t.Fatalf("missing route %s", action)
		}
	}
	if !routes["GET /api/activity/v1/records/:kind"] {
		t.Fatal("missing records")
	}
	t.Setenv("INTERNAL_TOKEN", strings.Repeat("t", 32))
	req := httptest.NewRequest("POST", "/internal/activity/v1/encoding", strings.NewReader(`{"edition_id":"8"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", strings.Repeat("t", 32))
	rec := httptest.NewRecorder()
	g.ServeHTTP(rec, req)
	if rec.Code != 200 || s.action != "encoding" || s.edition != "8" {
		t.Fatalf("dispatch %d %s", rec.Code, rec.Body.String())
	}
}
func TestBusinessErrorsMapToExistingHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{{fault.New("INVALID_DRAW_COUNT"), 400, "INVALID_DRAW_COUNT"}, {fault.New("ROUND_FULL"), 409, "ROUND_FULL"}, {fault.New("OPERATION_RESULT_UNKNOWN"), 503, "OPERATION_RESULT_UNKNOWN"}, {fault.ErrNotFound, 404, "NOT_FOUND"}, {errors.New("connection down"), 503, "DEPENDENCY_UNAVAILABLE"}} {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		Respond(ctx, nil, tc.err)
		var body map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if rec.Code != tc.status || body["error"] != tc.code {
			t.Fatalf("got %d %s", rec.Code, rec.Body.String())
		}
	}
}
func TestBindingStaysInHTTPDTO(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		raw   string
		valid bool
	}{{`{}`, false}, {`{"edition_id":"8","count":10}`, true}, {`{"edition_id":"8","count":"ten"}`, false}} {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest("POST", "/", strings.NewReader(tc.raw))
		ctx.Request.Header.Set("Content-Type", "application/json")
		var dto Request
		err := ctx.ShouldBindJSON(&dto)
		if (err == nil) != tc.valid {
			t.Fatalf("binding %s: %v", tc.raw, err)
		}
	}
}
