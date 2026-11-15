package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"

	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/domain"
	cache "activity/services/ranking-service/internal/ranking/infrastructure/redis"
)

type liveEdition struct {
	// The live leaderboard's active instance serializes writes and rebuilds per edition.
	writer     chan struct{}
	generation atomic.Pointer[cache.Generation]
	retired    bool      // protected by writer
	retryAfter time.Time // only used by MaintainCache
}

func (r *Repository) edition(id string) *liveEdition {
	if state, ok := r.editions.Load(id); ok {
		return state.(*liveEdition)
	}
	state, _ := r.editions.LoadOrStore(id, &liveEdition{writer: make(chan struct{}, 1)})
	return state.(*liveEdition)
}

func (s *liveEdition) lock(ctx context.Context) error {
	select {
	case s.writer <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *liveEdition) unlock() { <-s.writer }

func validateFact(kind string, f domain.Fact) error {
	for _, id := range []string{f.Edition, f.User} {
		if !canonicalID(id) {
			return platform.Fail(409, "INVALID_EVENT_ID")
		}
	}
	switch kind {
	case "ParticipationInitialized.v1":
		if f.Draw != "" {
			return platform.Fail(409, "INVALID_INITIALIZATION")
		}
		_, err := number(f.Created)
		return err
	case "FormalDrawCommitted.v1":
		if !canonicalID(f.Draw) {
			return platform.Fail(409, "INVALID_DRAW")
		}
		seq, seqErr := number(f.Seq)
		delta, deltaErr := number(f.Consumed)
		_, atErr := number(f.Occurred)
		if seqErr != nil || deltaErr != nil || atErr != nil || seq == 0 || delta == 0 {
			return platform.Fail(409, "INVALID_DRAW_FACT")
		}
		return nil
	default:
		return platform.Fail(409, "INVALID_EVENT_TYPE")
	}
}

func canonicalID(id string) bool {
	value, err := platform.UID(id)
	return err == nil && strconv.FormatUint(value, 10) == id
}

func (r *Repository) Consume(ctx context.Context, ev platform.Event) error {
	if ev.Type == "RankingAwardGranted.v1" {
		return r.receipt(ctx, ev)
	}
	var f domain.Fact
	if err := json.Unmarshal(ev.Payload, &f); err != nil {
		return platform.Fail(409, "INVALID_EVENT")
	}
	if err := validateFact(ev.Type, f); err != nil {
		return err
	}
	s := r.edition(f.Edition)
	if err := s.lock(ctx); err != nil {
		return err
	}
	g := s.generation.Load()
	if g != nil {
		op, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
		err := r.Cache.Apply(op, g, f)
		cancel()
		if err != nil {
			s.generation.CompareAndSwap(g, nil)
			slog.Warn("ranking cache write deferred", "edition_id", f.Edition, "error", err)
		}
	}
	err := r.storeFact(ctx, f)
	var fault *platform.Fault
	if g != nil && errors.As(err, &fault) && fault.Status == 409 {
		s.generation.CompareAndSwap(g, nil)
	}
	s.unlock()
	if err != nil || f.Hourly == nil || r.ProjectHourly == nil {
		return err
	}
	op, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	if e := r.ProjectHourly(op, f); e != nil && ctx.Err() == nil {
		// The source outbox remains replayable after ACK.
		slog.Debug("hourly ranking projection deferred", "edition_id", f.Edition, "cutoff_sec", f.Hourly.Cutoff, "error", e)
	}
	return nil
}

// MaintainCache is independent of queue ACKs and retries unavailable editions.
func (r *Repository) MaintainCache(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	discovered := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !discovered {
				op, cancel := context.WithTimeout(ctx, 2*time.Second)
				err := r.discoverEditions(op)
				cancel()
				discovered = err == nil
				if err != nil && ctx.Err() == nil {
					slog.Warn("ranking cache edition discovery deferred", "error", err)
				}
			}
			r.editions.Range(func(key, value any) bool {
				s := value.(*liveEdition)
				if s.generation.Load() == nil && !time.Now().Before(s.retryAfter) {
					op, cancel := context.WithTimeout(ctx, 30*time.Second)
					err := r.rebuild(op, key.(string), s)
					cancel()
					if err != nil && ctx.Err() == nil {
						s.retryAfter = time.Now().Add(5 * time.Second)
						slog.Warn("ranking cache rebuild deferred", "edition_id", key, "error", err)
					}
				}
				return ctx.Err() == nil
			})
		}
	}
}

func (r *Repository) discoverEditions(ctx context.Context) error {
	rows, err := r.DB.QueryContext(ctx, `SELECT DISTINCT CAST(e.edition_id AS CHAR)
		FROM ranking_entry e LEFT JOIN ranking_settlement s ON s.edition_id=e.edition_id
		WHERE s.edition_id IS NULL OR s.status='PENDING'
		UNION SELECT CAST(edition_id AS CHAR) FROM ranking_settlement WHERE status IN ('AWARDING','COMPLETED')`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var edition string
		if err := rows.Scan(&edition); err != nil {
			return err
		}
		r.edition(edition)
	}
	return rows.Err()
}

func (r *Repository) rebuild(ctx context.Context, edition string, s *liveEdition) error {
	if r.Cache == nil {
		return nil
	}
	select {
	case s.writer <- struct{}{}:
		defer s.unlock()
	default:
		return nil
	}
	if s.generation.Load() != nil || s.retired {
		return nil
	}
	started := time.Now()
	tx, err := r.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	phase, err := settlementPhase(ctx, tx, edition)
	if err != nil {
		return err
	}
	if phase != "" && phase != domain.SettlementPending {
		if phase == domain.SettlementAwarding || phase == domain.SettlementCompleted {
			err = r.Cache.Retire(ctx, edition)
			s.retired = err == nil
		}
		return err
	}
	start, cancel := context.WithTimeout(ctx, 2*time.Second)
	g, err := r.Cache.Start(start, edition)
	cancel()
	if err != nil {
		return err
	}

	// Zero-score users without any draws use their participation time.
	// All draw facts, even beyond a sequence gap, feed both the score and Set.
	rows, err := tx.QueryContext(ctx, `SELECT CAST(e.user_id AS CHAR),CAST(e.score_reached_at_sec AS CHAR),e.score
		FROM ranking_entry e WHERE e.edition_id=? AND NOT EXISTS (
			SELECT 1 FROM score_fact f WHERE f.edition_id=e.edition_id AND f.user_id=e.user_id)`, edition)
	if err != nil {
		return err
	}
	err = r.replayRows(ctx, g, rows, func(rows *sql.Rows, f *domain.Fact) error {
		var score int64
		if err := rows.Scan(&f.User, &f.Created, &score); err != nil {
			return err
		}
		if score != 0 {
			return errors.New("ranking entry has score without draw facts")
		}
		return nil
	})
	if err != nil {
		return err
	}
	rows, err = tx.QueryContext(ctx, `SELECT CAST(user_id AS CHAR),CAST(draw_id AS CHAR),
		CAST(consumed_petals AS CHAR),CAST(occurred_at_sec AS CHAR) FROM score_fact WHERE edition_id=?`, edition)
	if err != nil {
		return err
	}
	if err = r.replayRows(ctx, g, rows, func(rows *sql.Rows, f *domain.Fact) error {
		return rows.Scan(&f.User, &f.Draw, &f.Consumed, &f.Occurred)
	}); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if _, err = r.Cache.Board(ctx, g, "1"); err != nil {
		return err
	}
	s.generation.Store(g)
	slog.Info("ranking cache rebuilt", "edition_id", edition, "elapsed", time.Since(started))
	return nil
}

func (r *Repository) replayRows(ctx context.Context, g *cache.Generation, rows *sql.Rows, scan func(*sql.Rows, *domain.Fact) error) error {
	defer rows.Close()
	batch := make([]domain.Fact, 0, 128)
	for rows.Next() {
		var f domain.Fact
		if err := scan(rows, &f); err != nil {
			return err
		}
		batch = append(batch, f)
		if len(batch) == cap(batch) {
			if err := r.Cache.ApplyBatch(ctx, g, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(batch) == 0 {
		return nil
	}
	return r.Cache.ApplyBatch(ctx, g, batch)
}
