package mysql

import (
	"context"
	"database/sql"
	"errors"

	"activity/services/identity-service/internal/identity/domain"
)

type Repository struct{ DB *sql.DB }

func (r Repository) Register(ctx context.Context, account domain.Account, session domain.Session, eventID uint64, payload []byte) error {
	if account.UserID == 0 || eventID == 0 || session.UserID != account.UserID || session.ID == "" || account.Status != "ACTIVE" {
		return errors.New("invalid registration")
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO account (user_id,account_name,password_hash,nickname,status) VALUES (?,?,?,?,?)`, account.UserID, account.Name, account.PasswordHash, account.Nickname, account.Status)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO login_session (session_id,user_id,refresh_token_hash,expires_at) VALUES (?,?,?,?)`, session.ID, session.UserID, session.RefreshHash[:], session.ExpiresAt.UTC())
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO outbox_event (event_id,event_type,aggregate_id,user_id,payload) VALUES (?,'AccountCreated.v1',?,?,?)`, eventID, account.UserID, account.UserID, payload)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r Repository) AccountByName(ctx context.Context, name string) (domain.Account, error) {
	var a domain.Account
	err := r.DB.QueryRowContext(ctx, `SELECT user_id,account_name,nickname,password_hash,status FROM account WHERE account_name=?`, name).Scan(&a.UserID, &a.Name, &a.Nickname, &a.PasswordHash, &a.Status)
	return a, err
}

func (r Repository) SessionAccount(ctx context.Context, id string) (domain.Session, domain.Account, error) {
	var s domain.Session
	var a domain.Account
	var hash []byte
	var revoked sql.NullTime
	err := r.DB.QueryRowContext(ctx, `SELECT s.session_id,s.user_id,s.refresh_token_hash,s.expires_at,s.revoked_at,a.user_id,a.account_name,a.nickname,a.password_hash,a.status FROM login_session s JOIN account a ON a.user_id=s.user_id WHERE s.session_id=?`, id).Scan(&s.ID, &s.UserID, &hash, &s.ExpiresAt, &revoked, &a.UserID, &a.Name, &a.Nickname, &a.PasswordHash, &a.Status)
	if err != nil {
		return s, a, err
	}
	if len(hash) != len(s.RefreshHash) {
		return s, a, errors.New("invalid stored refresh hash")
	}
	copy(s.RefreshHash[:], hash)
	if revoked.Valid {
		s.RevokedAt = &revoked.Time
	}
	return s, a, nil
}
