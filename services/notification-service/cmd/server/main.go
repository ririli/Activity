package main

import (
	"activity/pkg/platform"
	"activity/pkg/ratelimit"
	"activity/services/notification-service/internal/notification/application"
	"activity/services/notification-service/internal/notification/infrastructure"
	httpapi "activity/services/notification-service/internal/notification/interfaces/http"
	"context"
	"log/slog"
	"time"
)

func main() {
	config, err := infrastructure.HubConfig()
	if err != nil {
		panic(err)
	}
	r := platform.Redis()
	defer r.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h, err := application.New(ctx, config, httpapi.EncodeFrame)
	if err != nil {
		panic(err)
	}
	defer h.Close()
	limiter := ratelimit.NewRedis()
	defer limiter.Close()
	g := platform.Router(nil, r)
	httpapi.Routes(g, h, limiter)
	platform.Run(g, func(ctx context.Context) {
		stop := context.AfterFunc(ctx, h.Close)
		defer stop()
		defer h.Close()
		go report(ctx, h)
		infrastructure.Subscribe(ctx, r, h)
	})
}

func report(ctx context.Context, h *application.Hub) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s := h.Stats()
			slog.Info("notification hub", "connections", s.Connections, "shard_queue_depths", s.ShardQueueDepths, "invalid_total", s.Invalid, "shard_dropped_total", s.ShardDropped, "connection_dropped_total", s.ConnectionDropped, "expired_shard_frames_total", s.Expired)
		}
	}
}
