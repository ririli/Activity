package application

import (
	"activity/pkg/hourly"
	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/domain"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type HourlyTask struct {
	hourly.Barrier
	Owner     string
	Watermark *hourly.Watermark
	Retries   int
}

type HourlyAward struct {
	hourly.Grant
	Owner   string
	Retries int
}

type HourlyStore interface {
	ScheduleHourly(context.Context, []hourly.Edition) (time.Time, error)
	ReceiveBarrier(context.Context, platform.Event) error
	ClaimHourly(context.Context) (*HourlyTask, error)
	SaveHourlyWatermark(context.Context, HourlyTask, hourly.Watermark) error
	PreviousHourlyWinners(context.Context, HourlyTask) ([]domain.Entry, error)
	FreezeHourly(context.Context, HourlyTask, []domain.Entry) error
	RetryHourly(context.Context, HourlyTask, bool) error
	HourlyRetirements(context.Context) ([]hourly.Window, error)
	FinishHourlyRetirement(context.Context, hourly.Window) error
	ClaimHourlyAward(context.Context) (*HourlyAward, error)
	FinishHourlyAward(context.Context, HourlyAward, string) error
	RetryHourlyAward(context.Context, HourlyAward) error
}

type HourlyActivity interface {
	Editions(context.Context) ([]hourly.Edition, error)
	Seal(context.Context, hourly.Window) (hourly.Watermark, error)
	Replay(context.Context, hourly.Replay) ([]platform.Event, error)
	Positions(context.Context, hourly.Positions) ([hourly.Lanes]int64, error)
	Grant(context.Context, hourly.Grant) (hourly.Receipt, error)
}

type HourlyProjection interface {
	Positions(context.Context, hourly.Window, int) ([hourly.Lanes]int64, error)
	Apply(context.Context, hourly.Window, int, int, []domain.Fact) (int64, error)
	Freeze(context.Context, hourly.Window, int, [hourly.Lanes]int64) ([]domain.Entry, error)
	Retire(context.Context, hourly.Window, int) error
}

type HourlyWorker struct {
	Store      HourlyStore
	Activity   HourlyActivity
	Projection HourlyProjection
	Wake       chan struct{}
}

func (w HourlyWorker) Barrier(ctx context.Context, event platform.Event) error {
	if err := w.Store.ReceiveBarrier(ctx, event); err != nil {
		return err
	}
	select {
	case w.Wake <- struct{}{}:
	default:
	}
	return nil
}

// Project keeps each open window current even when committed messages overtake
// each other. The settlement worker independently reconciles remaining gaps.
func (w HourlyWorker) Project(ctx context.Context, f domain.Fact) error {
	p := f.Hourly
	if p == nil {
		return nil
	}
	if p.Shard < 0 || p.Shard >= hourly.Shards || p.Lane < 0 || p.Lane >= hourly.Lanes {
		return fmt.Errorf("invalid hourly routing")
	}
	window := hourly.Window{Edition: f.Edition, Cutoff: p.Cutoff}
	last, err := w.Projection.Apply(ctx, window, p.Shard, p.Lane, []domain.Fact{f})
	if err != nil || last >= p.Seq || last < 0 {
		return err
	}
	_, err = w.replay(ctx, hourly.Replay{Window: window, Shard: p.Shard, Lane: p.Lane, After: last, Through: p.Seq})
	return err
}

// Schedule discovers new editions periodically and also wakes at the next cutoff.
func (w HourlyWorker) Schedule(ctx context.Context) {
	for ctx.Err() == nil {
		op, cancel := context.WithTimeout(ctx, 10*time.Second)
		editions, err := w.Activity.Editions(op)
		var next time.Time
		if err == nil {
			next, err = w.Store.ScheduleHourly(op, editions)
		}
		cancel()
		if err != nil && ctx.Err() == nil {
			slog.Warn("hourly barrier scheduling delayed", "error", err)
		}
		wait := 5 * time.Second
		if !next.IsZero() {
			wait = min(wait, max(100*time.Millisecond, time.Until(next)))
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
}

// Warm replays gaps while the window is open, including after a Redis restart.
func (w HourlyWorker) Warm(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for ctx.Err() == nil {
		op, cancel := context.WithTimeout(ctx, 10*time.Second)
		editions, err := w.Activity.Editions(op)
		if err == nil {
			now := time.Now()
			cutoff := hourly.NextCutoff(now)
			for _, edition := range editions {
				if now.Before(edition.Start) || !cutoff.Before(edition.End) {
					continue
				}
				w.warmWindow(op, hourly.Window{Edition: edition.ID, Cutoff: cutoff.Unix()})
			}
		} else if ctx.Err() == nil {
			slog.Warn("hourly open windows unavailable", "error", err)
		}
		cancel()
		select {
		case <-ctx.Done():
		case <-ticker.C:
		}
	}
}

func (w HourlyWorker) warmWindow(ctx context.Context, window hourly.Window) {
	var shards sync.WaitGroup
	for shard := range hourly.Shards {
		shards.Go(func() {
			mark, err := w.Activity.Positions(ctx, hourly.Positions{Window: window, Shard: shard})
			if err == nil {
				_, err = w.reconcileShard(ctx, window, shard, mark)
			}
			if err != nil && ctx.Err() == nil {
				slog.Debug("hourly open window reconciliation deferred", "edition_id", window.Edition, "shard", shard, "error", err)
			}
		})
	}
	shards.Wait()
}

func (w HourlyWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for ctx.Err() == nil {
		op, cancel := context.WithTimeout(ctx, 45*time.Second)
		err := w.SettleNext(op)
		cancel()
		if err != nil && ctx.Err() == nil {
			slog.Warn("hourly settlement delayed", "error", err)
		}
		select {
		case <-ctx.Done():
		case <-ticker.C:
		case <-w.Wake:
		}
	}
}

func (w HourlyWorker) Retire(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for ctx.Err() == nil {
		op, cancel := context.WithTimeout(ctx, 4*time.Second)
		windows, err := w.Store.HourlyRetirements(op)
		if err == nil {
			for _, window := range windows {
				if err := w.retireWindow(op, window); err != nil && ctx.Err() == nil {
					slog.Warn("hourly cache retirement delayed", "edition_id", window.Edition, "cutoff_sec", window.Cutoff, "error", err)
				}
			}
		} else if ctx.Err() == nil {
			slog.Warn("hourly cache retirement discovery delayed", "error", err)
		}
		cancel()
		select {
		case <-ctx.Done():
		case <-ticker.C:
		}
	}
}

func (w HourlyWorker) retireWindow(ctx context.Context, window hourly.Window) error {
	for shard := range hourly.Shards {
		if err := w.Projection.Retire(ctx, window, shard); err != nil {
			return err
		}
	}
	return w.Store.FinishHourlyRetirement(ctx, window)
}

func (w HourlyWorker) SettleNext(ctx context.Context) error {
	task, err := w.Store.ClaimHourly(ctx)
	if err != nil || task == nil {
		return err
	}
	if time.Since(time.Unix(task.Cutoff, 0)) > time.Minute {
		slog.Warn("hourly settlement exceeded 60 seconds", "edition_id", task.Edition, "cutoff_sec", task.Cutoff)
	}
	complete, err := w.freeze(ctx, *task)
	if complete {
		return err
	}
	// Use a fresh deadline to release the lease even when the processing step timed out.
	release, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	return errors.Join(err, w.Store.RetryHourly(release, *task, err != nil))
}

func (w HourlyWorker) freeze(ctx context.Context, task HourlyTask) (bool, error) {
	mark := task.Watermark
	if mark == nil {
		sealed, err := w.Activity.Seal(ctx, task.Window)
		if err != nil {
			return false, err
		}
		if err := w.Store.SaveHourlyWatermark(ctx, task, sealed); err != nil {
			return false, err
		}
		mark = &sealed
	}
	// One goroutine per Redis shard. Every lane gets a bounded replay page per step.
	type shardResult struct {
		entries []domain.Entry
		ready   bool
		err     error
	}
	results := make(chan shardResult, hourly.Shards)
	for shard := range hourly.Shards {
		go func() {
			entries, ready, err := w.freezeShard(ctx, task.Window, shard, mark[shard])
			results <- shardResult{entries, ready, err}
		}()
	}
	var candidates []domain.Entry
	ready := true
	var shardErr error
	for range hourly.Shards {
		result := <-results
		ready = ready && result.ready
		shardErr = errors.Join(shardErr, result.err)
		candidates = append(candidates, result.entries...)
	}
	if !ready || shardErr != nil {
		return false, shardErr
	}
	previous, err := w.Store.PreviousHourlyWinners(ctx, task)
	if err != nil {
		return false, err
	}
	winners := domain.Top(append(candidates, previous...), hourly.Winners)
	if err := w.Store.FreezeHourly(ctx, task, winners); err != nil {
		return false, err
	}
	slog.Info("hourly ranking frozen", "edition_id", task.Edition, "cutoff_sec", task.Cutoff, "winners", len(winners), "elapsed", time.Since(time.Unix(task.Cutoff, 0)))
	return true, nil
}

func (w HourlyWorker) freezeShard(ctx context.Context, window hourly.Window, shard int, mark [hourly.Lanes]int64) ([]domain.Entry, bool, error) {
	ready, err := w.reconcileShard(ctx, window, shard, mark)
	if !ready || err != nil {
		return nil, false, err
	}
	entries, err := w.Projection.Freeze(ctx, window, shard, mark)
	return entries, err == nil, err
}

func (w HourlyWorker) reconcileShard(ctx context.Context, window hourly.Window, shard int, mark [hourly.Lanes]int64) (bool, error) {
	positions, err := w.Projection.Positions(ctx, window, shard)
	if err != nil {
		return false, err
	}
	ready := true
	for lane, through := range mark {
		after := positions[lane]
		if after >= through {
			continue
		}
		last, err := w.replay(ctx, hourly.Replay{Window: window, Shard: shard, Lane: lane, After: after, Through: through})
		if err != nil {
			return false, err
		}
		ready = ready && last >= through
	}
	return ready, nil
}

func (w HourlyWorker) replay(ctx context.Context, request hourly.Replay) (int64, error) {
	page, err := w.Activity.Replay(ctx, request)
	if err != nil {
		return 0, err
	}
	if len(page) == 0 {
		return 0, fmt.Errorf("missing hourly facts: shard %d lane %d after %d", request.Shard, request.Lane, request.After)
	}
	facts := make([]domain.Fact, len(page))
	for i, ev := range page {
		if ev.Type != "FormalDrawCommitted.v1" && ev.Type != "ParticipationInitialized.v1" {
			return 0, fmt.Errorf("invalid hourly fact type %q", ev.Type)
		}
		if err := json.Unmarshal(ev.Payload, &facts[i]); err != nil {
			return 0, err
		}
		p := facts[i].Hourly
		if p == nil || p.Seq != request.After+int64(i)+1 || p.Seq > request.Through {
			return 0, fmt.Errorf("invalid hourly replay sequence")
		}
	}
	return w.Projection.Apply(ctx, request.Window, request.Shard, request.Lane, facts)
}

func (w HourlyWorker) Deliver(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for ctx.Err() == nil {
		found, err := w.DeliverNext(ctx)
		if err != nil && ctx.Err() == nil {
			slog.Warn("hourly award delivery delayed", "error", err)
		}
		if !found {
			select {
			case <-ctx.Done():
			case <-ticker.C:
			}
		}
	}
}

func (w HourlyWorker) DeliverNext(ctx context.Context) (bool, error) {
	op, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	award, err := w.Store.ClaimHourlyAward(op)
	if err != nil || award == nil {
		return false, err
	}
	if time.Since(time.Unix(award.Cutoff, 0)) > time.Minute {
		slog.Warn("hourly award still pending after 60 seconds", "edition_id", award.Edition, "cutoff_sec", award.Cutoff, "rank", award.Rank)
	}
	receipt, err := w.Activity.Grant(op, award.Grant)
	if err == nil {
		_, err = platform.UID(receipt.GrantID)
	}
	if err == nil {
		err = w.Store.FinishHourlyAward(op, *award, receipt.GrantID)
		if err == nil {
			return true, nil
		}
	}
	release, stop := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer stop()
	return true, errors.Join(err, w.Store.RetryHourlyAward(release, *award))
}
