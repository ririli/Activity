package mysql

import (
	"activity/pkg/platform"
	"context"
	"log/slog"
	"time"
)

func TryRanking(ctx context.Context, ev platform.Event) {
	if ev.Type != "FormalDrawCommitted.v1" && ev.Type != "ParticipationInitialized.v1" {
		return
	}
	op, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	var response any
	if e := platform.Call(op, platform.Env("RANKING_URL", "http://localhost:8083"), "/internal/ranking/v1/projection", ev, &response); e != nil {
		slog.Warn("ranking cache update deferred", "event_type", ev.Type, "error", e)
	}
}
