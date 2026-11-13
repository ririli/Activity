package activityhttp

import (
	"activity/pkg/hourly"
	"activity/pkg/platform"
	"context"
	"time"
)

type Client struct{ URL string }

func (c Client) Editions(ctx context.Context) ([]hourly.Edition, error) {
	var out []hourly.Edition
	err := platform.CallTimeout(ctx, c.URL, "/internal/activity/v1/hourly-editions", struct{}{}, &out, 3*time.Second)
	return out, err
}

func (c Client) Seal(ctx context.Context, w hourly.Window) (hourly.Watermark, error) {
	var out hourly.Watermark
	err := platform.CallTimeout(ctx, c.URL, "/internal/activity/v1/hourly-windows/seal", w, &out, 10*time.Second)
	return out, err
}

func (c Client) Replay(ctx context.Context, r hourly.Replay) ([]platform.Event, error) {
	var out []platform.Event
	err := platform.CallTimeout(ctx, c.URL, "/internal/activity/v1/hourly-windows/replay", r, &out, 3*time.Second)
	return out, err
}

func (c Client) Positions(ctx context.Context, p hourly.Positions) ([hourly.Lanes]int64, error) {
	var out [hourly.Lanes]int64
	err := platform.CallTimeout(ctx, c.URL, "/internal/activity/v1/hourly-windows/positions", p, &out, 3*time.Second)
	return out, err
}

func (c Client) Grant(ctx context.Context, g hourly.Grant) (hourly.Receipt, error) {
	var out hourly.Receipt
	err := platform.CallTimeout(ctx, c.URL, "/internal/activity/v1/hourly-ranking-grants", g, &out, 6*time.Second)
	return out, err
}
