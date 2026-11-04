package mysql

import (
	"activity/pkg/platform"
	"activity/services/identity-service/internal/identity/domain"
	"context"
	"crypto/subtle"
	"database/sql"
	"strings"
	"time"
)

func (r Repository) CreateSession(ctx context.Context, s domain.Session) error {
	_, e := r.DB.ExecContext(ctx, `INSERT INTO login_session(session_id,user_id,refresh_token_hash,expires_at) VALUES(?,?,?,?)`, s.ID, s.UserID, s.RefreshHash[:], s.ExpiresAt)
	return e
}
func (r Repository) Rotate(ctx context.Context, id string, old, newHash [32]byte) error {
	tx, e := r.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var h []byte
	var revoked sql.NullTime
	var expires time.Time
	var status string
	e = tx.QueryRowContext(ctx, `SELECT s.refresh_token_hash,s.revoked_at,s.expires_at,a.status FROM login_session s JOIN account a ON a.user_id=s.user_id WHERE s.session_id=? FOR UPDATE`, id).Scan(&h, &revoked, &expires, &status)
	if e != nil {
		return e
	}
	if revoked.Valid || !expires.After(time.Now()) || status != "ACTIVE" {
		return platform.Fail(401, "INVALID_SESSION")
	}
	if subtle.ConstantTimeCompare(h, old[:]) != 1 {
		_, e = tx.ExecContext(ctx, `UPDATE login_session SET revoked_at=UTC_TIMESTAMP(6) WHERE session_id=?`, id)
		if e != nil {
			return e
		}
		if e = tx.Commit(); e != nil {
			return e
		}
		return platform.Fail(401, "REFRESH_REPLAY")
	}
	_, e = tx.ExecContext(ctx, `UPDATE login_session SET refresh_token_hash=? WHERE session_id=?`, newHash[:], id)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (r Repository) Revoke(ctx context.Context, id string) error {
	_, e := r.DB.ExecContext(ctx, `UPDATE login_session SET revoked_at=COALESCE(revoked_at,UTC_TIMESTAMP(6)) WHERE session_id=?`, id)
	return e
}
func (r Repository) Profiles(ctx context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	args := make([]any, len(ids))
	marks := make([]string, len(ids))
	for i, id := range ids {
		if _, e := platform.UID(id); e != nil {
			return nil, e
		}
		args[i] = id
		marks[i] = "?"
	}
	rows, e := r.DB.QueryContext(ctx, `SELECT CAST(user_id AS CHAR),nickname FROM account WHERE user_id IN (`+strings.Join(marks, ",")+`)`, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var id, nick string
		if e = rows.Scan(&id, &nick); e != nil {
			return nil, e
		}
		out[id] = nick
	}
	return out, rows.Err()
}
