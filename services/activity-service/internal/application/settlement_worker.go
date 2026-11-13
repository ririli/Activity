package application

import (
	"activity/services/activity-service/internal/fault"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"
)

type SettlementTask struct {
	Edition    string
	Step       string
	Partition  int
	RetryCount int
	Cursor     []byte
	Owner      string
}

type TaskStore interface {
	Ensure(ctx context.Context) error
	Claim(ctx context.Context) (SettlementTask, bool, error)
	Complete(ctx context.Context, task SettlementTask, cursor []byte) error
	Release(ctx context.Context, task SettlementTask, cursor []byte, delay time.Duration) error
	Retry(ctx context.Context, task SettlementTask, cursor []byte, cause error, base time.Duration) error
	Fail(ctx context.Context, task SettlementTask, code string) error
	UsersPage(ctx context.Context, edition string, partition int, after uint64, limit int) ([]string, error)
	FailedCount(ctx context.Context, edition string) (int, error)
	RemainingCount(ctx context.Context, edition string) (int, error)
	CompleteEdition(ctx context.Context, edition string) error
}

type SettlementExecutor interface {
	SettleUser(context.Context, JoinCommand) (any, error)
	CloseEdition(context.Context, string) (any, error)
}

type RankingSettler interface {
	Settle(context.Context, string) (status string, pending int64, err error)
}

type Bootstrapper interface {
	Bootstrap(context.Context) error
}

type SettlementWorker struct {
	Store     TaskStore
	Exec      SettlementExecutor
	Ranking   RankingSettler
	Bootstrap Bootstrapper
	Batch     int
	Disabled  bool
}

func (w SettlementWorker) Tick(ctx context.Context) error {
	if err := w.Store.Ensure(ctx); err != nil {
		return err
	}
	task, ok, err := w.Store.Claim(ctx)
	if err != nil || !ok {
		return err
	}
	return w.execute(ctx, task)
}

func (w SettlementWorker) execute(ctx context.Context, task SettlementTask) error {
	switch task.Step {
	case "01_CLOSE_EDITION":
		if _, err := w.Exec.CloseEdition(ctx, task.Edition); err != nil {
			return w.Store.Retry(ctx, task, task.Cursor, err, 5*time.Second)
		}
		return w.Store.Complete(ctx, task, task.Cursor)
	case "02_SETTLE_USERS":
		return w.settleUsers(ctx, task)
	case "03_RANKING":
		return w.settleRanking(ctx, task)
	case "04_FINALIZE":
		return w.finalize(ctx, task)
	default:
		return w.Store.Fail(ctx, task, "UNKNOWN_SETTLEMENT_STEP")
	}
}

func (w SettlementWorker) batch() int {
	if w.Batch <= 0 || w.Batch > 500 {
		return 100
	}
	return w.Batch
}

func (w SettlementWorker) settleUsers(ctx context.Context, task SettlementTask) error {
	var cursor struct {
		After string `json:"after_user_id"`
	}
	if len(task.Cursor) > 0 {
		if err := json.Unmarshal(task.Cursor, &cursor); err != nil {
			return w.Store.Fail(ctx, task, "INVALID_SETTLEMENT_CURSOR")
		}
	}
	if cursor.After == "" {
		cursor.After = "0"
	}
	after, err := parseCursor(cursor.After)
	if err != nil {
		return w.Store.Fail(ctx, task, "INVALID_SETTLEMENT_CURSOR")
	}
	batch := w.batch()
	users, err := w.Store.UsersPage(ctx, task.Edition, task.Partition, after, batch)
	if err != nil {
		return w.Store.Retry(ctx, task, task.Cursor, err, 5*time.Second)
	}
	last := cursor.After
	for _, user := range users {
		if _, err = w.Exec.SettleUser(ctx, JoinCommand{Actor: Actor{User: user, Edition: task.Edition}}); err != nil {
			raw, _ := json.Marshal(map[string]string{"after_user_id": last})
			return w.Store.Retry(ctx, task, raw, err, 5*time.Second)
		}
		last = user
	}
	raw, _ := json.Marshal(map[string]string{"after_user_id": last})
	if len(users) < batch {
		return w.Store.Complete(ctx, task, raw)
	}
	return w.Store.Release(ctx, task, raw, time.Second)
}

func (w SettlementWorker) settleRanking(ctx context.Context, task SettlementTask) error {
	callCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	status, _, err := w.Ranking.Settle(callCtx, task.Edition)
	if err != nil {
		var failure *fault.Error
		if errors.As(err, &failure) && failure.Code == "RANKING_SETTLEMENT_FAILED" {
			return w.Store.Fail(ctx, task, "RANKING_SETTLEMENT_FAILED")
		}
		return w.Store.Retry(ctx, task, task.Cursor, err, 5*time.Second)
	}
	switch status {
	case "COMPLETED":
		return w.Store.Complete(ctx, task, task.Cursor)
	case "FAILED":
		return w.Store.Fail(ctx, task, "RANKING_SETTLEMENT_FAILED")
	default:
		return w.Store.Release(ctx, task, task.Cursor, 5*time.Second)
	}
}

func (w SettlementWorker) finalize(ctx context.Context, task SettlementTask) error {
	failed, err := w.Store.FailedCount(ctx, task.Edition)
	if err != nil {
		return w.Store.Retry(ctx, task, task.Cursor, err, 5*time.Second)
	}
	if failed > 0 {
		return w.Store.Fail(ctx, task, "PRECEDING_SETTLEMENT_STEP_FAILED")
	}
	remaining, err := w.Store.RemainingCount(ctx, task.Edition)
	if err != nil {
		return w.Store.Retry(ctx, task, task.Cursor, err, 5*time.Second)
	}
	if remaining > 0 {
		return w.Store.Release(ctx, task, task.Cursor, 5*time.Second)
	}
	if err = w.Store.CompleteEdition(ctx, task.Edition); err != nil {
		return w.Store.Retry(ctx, task, task.Cursor, err, 5*time.Second)
	}
	return w.Store.Complete(ctx, task, task.Cursor)
}

func parseCursor(value string) (uint64, error) {
	return strconv.ParseUint(value, 10, 63)
}
