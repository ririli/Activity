package mysql

import (
	"activity/pkg/platform"
	"context"
	"time"
)

func (r *Repository) Publish(ctx context.Context) {
	for i := 0; i < 2; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case b := <-r.Notify:
					op, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
					_ = r.Redis.Publish(op, "activity:"+platform.Env("APP_ENV", "local")+":notification:ring-awarded:v1", b).Err()
					cancel()
				}
			}
		}()
	}
	<-ctx.Done()
}
