package mysql

import (
	app "activity/services/activity-service/internal/application"
	"context"
)

func (q *Queries) BootstrapCandidates(ctx context.Context) ([]app.JoinCommand, error) {
	rows, err := q.DB.QueryContext(ctx, `SELECT CAST(w.user_id AS CHAR),CAST(e.edition_id AS CHAR) FROM wallet w JOIN activity_edition e ON e.status='ACTIVE' AND e.starts_at<=UTC_TIMESTAMP(6) AND e.ends_at>UTC_TIMESTAMP(6) LEFT JOIN user_participation p ON p.user_id=w.user_id AND p.edition_id=e.edition_id WHERE p.user_id IS NULL ORDER BY w.user_id,e.edition_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []app.JoinCommand{}
	for rows.Next() {
		var c app.JoinCommand
		if err = rows.Scan(&c.User, &c.Edition); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
