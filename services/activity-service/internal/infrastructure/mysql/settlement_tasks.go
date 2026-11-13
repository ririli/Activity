package mysql

import (
	"activity/pkg/platform"
	activityapp "activity/services/activity-service/internal/application"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type TaskStore struct{ DB *sql.DB }

func taskOwner() string {
	owner := fmt.Sprintf("%s-%d", platform.Env("NODE_ID", "activity"), os.Getpid())
	if len(owner) > 31 {
		owner = owner[:31]
	}
	return owner + "-" + platform.Random()
}

func (s TaskStore) Ensure(ctx context.Context) error {
	rows, err := s.DB.QueryContext(ctx, `SELECT CAST(edition_id AS CHAR) FROM activity_edition WHERE ends_at<=UTC_TIMESTAMP(6) AND settlement_status NOT IN ('COMPLETED','FAILED')`)
	if err != nil {
		return err
	}
	editions := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		editions = append(editions, id)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	type taskKey struct {
		step string
		part int
	}
	keys := []taskKey{{"01_CLOSE_EDITION", -1}}
	for i := 0; i < 4; i++ {
		keys = append(keys, taskKey{"02_SETTLE_USERS", i})
	}
	keys = append(keys, taskKey{"03_RANKING", -1}, taskKey{"04_FINALIZE", -1})
	for _, edition := range editions {
		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		for _, k := range keys {
			if _, err = tx.ExecContext(ctx, `INSERT INTO settlement_task(edition_id,step,partition_no) VALUES(?,?,?) ON DUPLICATE KEY UPDATE edition_id=edition_id`, edition, k.step, k.part); err != nil {
				tx.Rollback()
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, `UPDATE activity_edition SET settlement_status='RUNNING' WHERE edition_id=? AND settlement_status NOT IN ('COMPLETED','FAILED')`, edition); err != nil {
			tx.Rollback()
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (s TaskStore) Claim(ctx context.Context) (activityapp.SettlementTask, bool, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return activityapp.SettlementTask{}, false, err
	}
	defer tx.Rollback()
	var task activityapp.SettlementTask
	err = tx.QueryRowContext(ctx, `SELECT CAST(t.edition_id AS CHAR),t.step,t.partition_no,t.retry_count,COALESCE(t.cursor_json,JSON_OBJECT())
		FROM settlement_task t
		WHERE t.status IN ('PENDING','RUNNING')
		  AND (t.lease_until IS NULL OR t.lease_until<UTC_TIMESTAMP(6))
		  AND t.next_retry_at<=UTC_TIMESTAMP(6)
		  AND NOT EXISTS (
		      SELECT 1 FROM settlement_task p
		      WHERE p.edition_id=t.edition_id AND p.step<t.step AND p.status<>'COMPLETED'
		  )
		ORDER BY t.edition_id,t.step,t.partition_no
		LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&task.Edition, &task.Step, &task.Partition, &task.RetryCount, &task.Cursor)
	if errors.Is(err, sql.ErrNoRows) {
		return activityapp.SettlementTask{}, false, tx.Commit()
	}
	if err != nil {
		return activityapp.SettlementTask{}, false, err
	}
	task.Owner = taskOwner()
	res, err := tx.ExecContext(ctx, `UPDATE settlement_task
		SET status='RUNNING',lease_owner=?,lease_until=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 60 SECOND)
		WHERE edition_id=? AND step=? AND partition_no=? AND status IN ('PENDING','RUNNING')`, task.Owner, task.Edition, task.Step, task.Partition)
	if err != nil {
		return activityapp.SettlementTask{}, false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return activityapp.SettlementTask{}, false, err
	}
	if n != 1 {
		return activityapp.SettlementTask{}, false, tx.Commit()
	}
	return task, true, tx.Commit()
}

func (s TaskStore) Complete(ctx context.Context, task activityapp.SettlementTask, cursor []byte) error {
	res, err := s.DB.ExecContext(ctx, `UPDATE settlement_task
		SET status='COMPLETED',cursor_json=?,lease_owner=NULL,lease_until=NULL,next_retry_at=UTC_TIMESTAMP(6)
		WHERE edition_id=? AND step=? AND partition_no=? AND lease_owner=? AND status='RUNNING'`, cursor, task.Edition, task.Step, task.Partition, task.Owner)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil || n != 1 {
		return errors.New("settlement task lease lost")
	}
	return nil
}

func (s TaskStore) Release(ctx context.Context, task activityapp.SettlementTask, cursor []byte, delay time.Duration) error {
	res, err := s.DB.ExecContext(ctx, `UPDATE settlement_task
		SET status='PENDING',cursor_json=?,lease_owner=NULL,lease_until=NULL,next_retry_at=?
		WHERE edition_id=? AND step=? AND partition_no=? AND lease_owner=? AND status='RUNNING'`, cursor, time.Now().UTC().Add(delay), task.Edition, task.Step, task.Partition, task.Owner)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil || n != 1 {
		return errors.New("settlement task lease lost")
	}
	return nil
}

func (s TaskStore) Retry(ctx context.Context, task activityapp.SettlementTask, cursor []byte, cause error, base time.Duration) error {
	delay := base << min(task.RetryCount, 6)
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}
	msg := cause.Error()
	if len(msg) > 1024 {
		msg = msg[:1024]
	}
	_, _ = s.DB.ExecContext(ctx, `UPDATE settlement_task
		SET status='PENDING',cursor_json=?,retry_count=retry_count+1,next_retry_at=?,last_error=?,lease_owner=NULL,lease_until=NULL
		WHERE edition_id=? AND step=? AND partition_no=? AND lease_owner=? AND status='RUNNING'`, cursor, time.Now().UTC().Add(delay), msg, task.Edition, task.Step, task.Partition, task.Owner)
	return cause
}

func (s TaskStore) Fail(ctx context.Context, task activityapp.SettlementTask, code string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE settlement_task
		SET status='FAILED',last_error=?,lease_owner=NULL,lease_until=NULL
		WHERE edition_id=? AND step=? AND partition_no=? AND lease_owner=? AND status='RUNNING'`, code, task.Edition, task.Step, task.Partition, task.Owner)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil || n != 1 {
		return errors.New("settlement task lease lost")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE activity_edition SET settlement_status='FAILED' WHERE edition_id=? AND settlement_status<>'COMPLETED'`, task.Edition); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return platform.Fail(409, code)
}

func (s TaskStore) UsersPage(ctx context.Context, edition string, partition int, after uint64, limit int) ([]string, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT CAST(user_id AS CHAR) FROM user_participation
		WHERE edition_id=? AND MOD(user_id,4)=? AND user_id>? ORDER BY user_id LIMIT ?`, edition, partition, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		users = append(users, id)
	}
	return users, rows.Err()
}

func (s TaskStore) FailedCount(ctx context.Context, edition string) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM settlement_task WHERE edition_id=? AND status='FAILED'`, edition).Scan(&n)
	return n, err
}

func (s TaskStore) RemainingCount(ctx context.Context, edition string) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM settlement_task WHERE edition_id=? AND step<>'04_FINALIZE' AND status<>'COMPLETED'`, edition).Scan(&n)
	return n, err
}

func (s TaskStore) CompleteEdition(ctx context.Context, edition string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE activity_edition SET settlement_status='COMPLETED' WHERE edition_id=? AND settlement_status NOT IN ('COMPLETED','FAILED')`, edition)
	return err
}

func (s TaskStore) Batch() int {
	raw := platform.Env("SETTLEMENT_BATCH_SIZE", "")
	if raw == "" {
		return 100
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 || n > 500 {
		return 100
	}
	return n
}
