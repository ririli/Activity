package infrastructure

import (
	"activity/pkg/platform"
	"activity/services/notification-service/internal/notification/application"
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

func Subscribe(ctx context.Context, r *redis.Client, h *application.Hub) {
	channel := "activity:" + platform.Env("APP_ENV", "local") + ":notification:ring-awarded:v1"
	for ctx.Err() == nil {
		p := r.Subscribe(ctx, channel)
		_, e := p.Receive(ctx)
		if e == nil {
			h.SetReady(true)
			for ctx.Err() == nil {
				op, cancel := context.WithTimeout(ctx, 2*time.Second)
				e = r.Ping(op).Err()
				cancel()
				if e != nil {
					break
				}
				msg, e := p.ReceiveTimeout(ctx, time.Second)
				if e == nil {
					if m, ok := msg.(*redis.Message); ok {
						h.Receive([]byte(m.Payload))
					}
				} else if ctx.Err() != nil {
					break
				}
			}
		}
		h.SetReady(false)
		p.Close()
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}
