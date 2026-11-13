package ranking

import (
	"activity/pkg/platform"
	"activity/services/activity-service/internal/fault"
	"context"
	"errors"
	"time"
)

type Client struct{ URL string }

func (c Client) Settle(ctx context.Context, edition string) (string, int64, error) {
	base := c.URL
	if base == "" {
		base = platform.Env("RANKING_URL", "http://localhost:8083")
	}
	var out struct {
		Status        string `json:"status"`
		PendingAwards int64  `json:"pending_awards"`
	}
	err := platform.CallTimeout(ctx, base, "/internal/ranking/v1/settle", map[string]string{"edition_id": edition}, &out, 20*time.Second)
	var failure *platform.Fault
	if errors.As(err, &failure) && failure.Status == 409 {
		err = fault.New("RANKING_SETTLEMENT_FAILED")
	}
	return out.Status, out.PendingAwards, err
}
