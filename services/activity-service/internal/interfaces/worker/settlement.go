package worker

import (
	app "activity/services/activity-service/internal/application"
	"context"
	"log/slog"
	"time"
)

type Settlement struct{ Workflow app.SettlementWorker }

func (w Settlement) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			op, cancel := context.WithTimeout(ctx, 30*time.Second)
			if w.Workflow.Bootstrap != nil {
				_ = w.Workflow.Bootstrap.Bootstrap(op)
			}
			if !w.Workflow.Disabled {
				if err := w.Workflow.Tick(op); err != nil {
					slog.Warn("settlement tick failed", "error", err)
				}
			}
			cancel()
		}
	}
}
