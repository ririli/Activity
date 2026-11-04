package mysql

import (
	"activity/pkg/platform"
	"activity/services/activity-service/internal/appstate"
	"context"
	"fmt"
	"time"
)

func (r *Repository) Bootstrap(ctx context.Context) error {
	rows, e := r.DB.QueryContext(ctx, `SELECT CAST(user_id AS CHAR) FROM wallet ORDER BY user_id`)
	if e != nil {
		return e
	}
	users := []string{}
	for rows.Next() {
		var u string
		if e = rows.Scan(&u); e != nil {
			rows.Close()
			return e
		}
		users = append(users, u)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	rows, e = r.DB.QueryContext(ctx, `SELECT CAST(edition_id AS CHAR) FROM activity_edition WHERE status='ACTIVE' AND starts_at<=UTC_TIMESTAMP(6) AND ends_at>UTC_TIMESTAMP(6)`)
	if e != nil {
		return e
	}
	editions := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return e
		}
		editions = append(editions, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, u := range users {
		for _, edition := range editions {
			var n int
			if e = r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_participation WHERE user_id=? AND edition_id=?`, u, edition).Scan(&n); e != nil {
				return e
			}
			if n == 0 {
				if _, e = r.App.Execute(ctx, u, appstate.Command{Action: "join", Edition: edition}); e != nil {
					return e
				}
			}
		}
	}
	return nil
}

func (r *Repository) Internal(ctx context.Context, action, edition, user, after string) (any, error) {
	if _, e := platform.UID(edition); e != nil {
		return nil, e
	}
	switch action {
	case "encoding":
		var start, end time.Time
		e := r.DB.QueryRowContext(ctx, `SELECT starts_at,ends_at FROM activity_edition WHERE edition_id=?`, edition).Scan(&start, &end)
		return map[string]int64{"min_sec": start.Unix(), "end_sec": end.Unix()}, e
	case "manifest":
		if after == "" {
			after = "0"
		}
		return queryMaps(ctx, r.DB, `SELECT CAST(user_id AS CHAR) user_id,CAST(draw_seq AS CHAR) draw_seq,CAST(ranking_score AS CHAR) total_score,CAST(ranking_score_reached_at_sec AS CHAR) score_reached_at_sec,CAST(FLOOR(UNIX_TIMESTAMP(created_at)) AS UNSIGNED) participation_created_at_sec FROM user_participation WHERE edition_id=? AND user_id>? ORDER BY user_id LIMIT 100`, edition, after)
	case "facts":
		u, e := platform.UID(user)
		if e != nil {
			return nil, e
		}
		if after == "" {
			after = "0"
		}
		return queryMaps(ctx, r.DB, fmt.Sprintf(`SELECT CAST(user_id AS CHAR) user_id,CAST(edition_id AS CHAR) edition_id,CAST(draw_id AS CHAR) draw_id,CAST(draw_seq AS CHAR) draw_seq,CAST(consumed_petals AS CHAR) consumed_petals,CAST(FLOOR(UNIX_TIMESTAMP(occurred_at)) AS CHAR) occurred_at_sec FROM %s WHERE user_id=? AND edition_id=? AND draw_seq>? ORDER BY draw_seq LIMIT 100`, platform.Shard("draw_order", u)), user, edition, after)
	}
	return nil, platform.Fail(400, "INVALID_INTERNAL_ACTION")
}
