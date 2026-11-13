package infrastructure

import (
	"activity/pkg/platform"
	"activity/services/notification-service/internal/notification/application"
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	"net"
	"time"
)

func Subscribe(ctx context.Context, r *redis.Client, h *application.Hub) {
	channel := "activity:" + platform.Env("APP_ENV", "local") + ":notification:ring-awarded:v1"
	for ctx.Err() == nil {
		p := r.Subscribe(ctx, channel)
		_, e := p.Receive(ctx)
		if e == nil {
			h.SetReady(true)
			var nextProbe time.Time
			for ctx.Err() == nil {
				if !time.Now().Before(nextProbe) {
					op, cancel := context.WithTimeout(ctx, 2*time.Second)
					e = r.Ping(op).Err()
					cancel()
					if e != nil {
						break
					}
					nextProbe = time.Now().Add(time.Second)
				}
				msg, e := p.ReceiveTimeout(ctx, time.Second)
				if e == nil {
					if m, ok := msg.(*redis.Message); ok {
						h.Receive([]byte(m.Payload))
					}
				} else {
					var timeout net.Error
					if ctx.Err() != nil || !errors.As(e, &timeout) || !timeout.Timeout() {
						break
					}
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
