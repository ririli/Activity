package mysql

import (
	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/domain"
	"context"
	"encoding/json"
)

// CacheUpdate accepts the same immutable draw delta as the outbox consumer.
func (r *Repository) CacheUpdate(ctx context.Context, ev platform.Event) error {
	return r.cacheUpdate(ctx, ev, true)
}
func (r *Repository) cacheUpdate(ctx context.Context, ev platform.Event, checkHistory bool) error {
	if r.Cache == nil {
		return nil
	}
	if ev.Type != "ParticipationInitialized.v1" && ev.Type != "FormalDrawCommitted.v1" {
		return platform.Fail(409, "INVALID_CACHE_EVENT")
	}
	if checkHistory {
		var f domain.Fact
		if e := json.Unmarshal(ev.Payload, &f); e != nil {
			return e
		}
		present, e := r.Cache.HasUser(ctx, f.Edition, f.User)
		if e != nil {
			return e
		}
		if !present {
			var score int64
			e = r.DB.QueryRowContext(ctx, `SELECT COALESCE(MAX(score),0) FROM ranking_entry WHERE edition_id=? AND user_id=?`, f.Edition, f.User).Scan(&score)
			if e != nil {
				return e
			}
			if score > 0 {
				return r.Cache.Disable(ctx, f.Edition)
			}
		}
	}
	return r.Cache.Apply(ctx, ev)
}
