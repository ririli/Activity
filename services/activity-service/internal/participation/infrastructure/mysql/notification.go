package mysql

import (
	"activity/pkg/platform"
	"activity/services/activity-service/internal/participation/domain"
	"context"
	"fmt"
	"time"
)

func (s *state) notify(p domain.Prize, grant uint64, source string) {
	if s.nickname == "" || len(p.Choices) > 0 || p.ID == 1001 || p.ID == 1002 {
		return
	}
	s.messages = append(s.messages, map[string]any{"schema_version": 1, "event_id": fmt.Sprintf("ring:%d:%s:v1", grant, source), "event_type": "ring_awarded", "occurred_at": s.now.UTC(), "expires_at": s.now.Add(30 * time.Second).UTC(), "edition_id": fmt.Sprint(s.edition), "source": source, "display_name": s.nickname, "item_id": p.ID, "item_name": p.Name, "quantity": p.Quantity})
}
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
