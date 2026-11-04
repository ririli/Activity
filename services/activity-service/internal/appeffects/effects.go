package appeffects

import (
	"activity/pkg/platform"
	"context"
	"log/slog"
	"time"
)

type Effects struct {
	Notify chan []byte
}

func (e Effects) Project(ev platform.Event) {
	if ev.Type != "FormalDrawCommitted.v1" && ev.Type != "ParticipationInitialized.v1" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	var response any
	if err := platform.Call(ctx, platform.Env("RANKING_URL", "http://localhost:8083"), "/internal/ranking/v1/projection", ev, &response); err != nil {
		slog.Warn("ranking cache update deferred", "event_type", ev.Type, "error", err)
	}
}

func (e Effects) Enqueue(messages []map[string]any) {
	if e.Notify == nil {
		return
	}
	for _, message := range messages {
		select {
		case e.Notify <- platform.JSON(message):
		default:
		}
	}
}
