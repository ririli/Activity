package application

import (
	assetsapp "activity/services/activity-service/internal/assets/application"
	"activity/services/activity-service/internal/fault"
	lotteryapp "activity/services/activity-service/internal/lottery/application"
	partapp "activity/services/activity-service/internal/participation/application"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

type Service struct {
	UoW     UnitOfWork
	Lock    UserLock
	Effects Effects
	Queries Queries
	Runtime Runtime
}

var operationPattern = regexp.MustCompile(`^[A-Za-z0-9_:-]{1,64}$`)

func uid(v string) (uint64, error) {
	id, err := strconv.ParseUint(v, 10, 63)
	if err != nil || id == 0 {
		return 0, fault.New("INVALID_ID")
	}
	return id, nil
}
func (s Service) newWork(ctx context.Context, tx Session, scope Scope, nickname string) *work {
	newID := func() (uint64, error) { return s.Runtime.NewID(ctx) }
	return &work{session: tx, scope: scope, nickname: nickname, runtime: s.Runtime, newID: newID, assets: assetsapp.Service{Repo: tx.Assets, NewID: newID}, lottery: lotteryapp.Service{Repo: tx.Lottery, Random: s.Runtime.Random, Now: s.Runtime.Now}, participation: partapp.Service{Repo: tx.Participation}}
}
func (s Service) mutate(ctx context.Context, a Actor, active bool, fn func(*work) (any, error)) (any, error) {
	return s.mutateWithRanking(ctx, a, active, false, fn)
}

func (s Service) mutateWithRanking(ctx context.Context, a Actor, active, scoring bool, fn func(*work) (any, error)) (any, error) {
	user, err := uid(a.User)
	if err != nil {
		return nil, err
	}
	edition, err := uid(a.Edition)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()
	if s.Lock == nil {
		return nil, fault.New("DEPENDENCY_UNAVAILABLE")
	}
	release, err := s.Lock.Acquire(ctx, a.User)
	if err != nil {
		return nil, err
	}
	defer release()
	biz, err := s.Runtime.NewID(ctx)
	if err != nil {
		return nil, err
	}
	scope := Scope{User: user, Edition: edition, Biz: biz}
	var out any
	var completed *work
	err = s.UoW.Within(ctx, scope, func(tx Session) error {
		w := s.newWork(ctx, tx, scope, a.DisplayName)
		if e := w.Load(active, scoring); e != nil {
			return e
		}
		var e error
		out, e = fn(w)
		if e == nil {
			completed = w
		}
		return e
	})
	if errors.Is(err, fault.ErrCommitUnknown) {
		return nil, fault.New("OPERATION_RESULT_UNKNOWN")
	}
	if err != nil {
		return nil, err
	}
	s.after(completed)
	return out, nil
}
func (s Service) after(w *work) {
	if s.Effects == nil || w == nil {
		return
	}
	s.Effects.Enqueue(w.messages)
}
func (s Service) Join(ctx context.Context, c JoinCommand) (any, error) {
	return s.mutate(ctx, c.Actor, true, func(w *work) (any, error) { return w.Summary(), nil })
}
func (s Service) Purchase(ctx context.Context, c PurchaseCommand) (any, error) {
	if !operationPattern.MatchString(c.Operation) {
		return nil, fault.New("INVALID_OPERATION_ID")
	}
	return s.mutate(ctx, c.Actor, true, func(w *work) (any, error) {
		return purchase(w, c)
	})
}
func (s Service) ClaimStage(ctx context.Context, c StageCommand) (any, error) {
	return s.mutate(ctx, c.Actor, true, func(w *work) (any, error) { return claimStage(w, c.Stage, "USER") })
}
func (s Service) OpenChest(ctx context.Context, c ChestCommand) (any, error) {
	return s.mutate(ctx, c.Actor, true, openChest)
}
func (s Service) ConfirmChest(ctx context.Context, c ChestCommand) (any, error) {
	return s.mutate(ctx, c.Actor, false, func(w *work) (any, error) { return confirmChest(w, c.ID) })
}
func (s Service) SelectReward(ctx context.Context, c SelectRewardCommand) (any, error) {
	return s.mutate(ctx, c.Actor, false, func(w *work) (any, error) { return selectReward(w, c.ID, c.Item) })
}
func (s Service) NextRound(ctx context.Context, c RoundCommand) (any, error) {
	return s.mutate(ctx, c.Actor, true, func(w *work) (any, error) { return nextRound(w, c.Round) })
}
func (s Service) Preview(ctx context.Context, c PreviewCommand) (any, error) {
	return s.mutate(ctx, c.Actor, true, preview)
}
func (s Service) ConfirmPreview(ctx context.Context, c PreviewCommand) (any, error) {
	return s.mutateWithRanking(ctx, c.Actor, true, true, confirmPreview)
}
func (s Service) AbandonPreview(ctx context.Context, c PreviewCommand) (any, error) {
	return s.mutate(ctx, c.Actor, true, abandonPreview)
}
func (s Service) SettleUser(ctx context.Context, c JoinCommand) (any, error) {
	return s.mutate(ctx, c.Actor, false, func(w *work) (any, error) {
		if err := w.edition.CheckEnded(); err != nil {
			return nil, err
		}
		if err := settleUser(w); err != nil {
			return nil, err
		}
		return map[string]string{"status": "COMPLETED"}, nil
	})
}
func (s Service) Draw(ctx context.Context, c DrawCommand) (any, error) {
	if !operationPattern.MatchString(c.Operation) {
		return nil, fault.New("INVALID_OPERATION_ID")
	}
	if c.Count != 1 && c.Count != 10 && c.Count != 30 {
		return nil, fault.New("INVALID_DRAW_COUNT")
	}
	user, err := uid(c.User)
	if err != nil {
		return nil, err
	}
	edition, err := uid(c.Edition)
	if err != nil {
		return nil, err
	}
	scope := Scope{User: user, Edition: edition}
	cmd := lotteryapp.DrawCommand{Edition: c.Edition, Operation: c.Operation, Pool: c.Pool, Count: c.Count}
	var out any
	var found bool
	err = s.UoW.Read(ctx, scope, func(tx Session) error {
		var e error
		out, found, e = s.newWork(ctx, tx, scope, "").Recover(cmd)
		return e
	})
	if err != nil || found {
		return out, err
	}
	// The second check is done before active-state loading so successful replay
	// remains available after the edition closes or progress changes.
	ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()
	if s.Lock == nil {
		return nil, fault.New("DEPENDENCY_UNAVAILABLE")
	}
	release, err := s.Lock.Acquire(ctx, c.User)
	if err != nil {
		return nil, err
	}
	defer release()
	scope.Biz, err = s.Runtime.NewID(ctx)
	if err != nil {
		return nil, err
	}
	var completed *work
	err = s.UoW.Within(ctx, scope, func(tx Session) error {
		w := s.newWork(ctx, tx, scope, c.DisplayName)
		var e error
		out, found, e = w.Recover(cmd)
		if e != nil || found {
			return e
		}
		if e = w.Load(true, true); e != nil {
			return e
		}
		out, e = draw(w, cmd, nil, 0)
		if e == nil {
			completed = w
		}
		return e
	})
	if errors.Is(err, fault.ErrCommitUnknown) {
		return nil, fault.New("OPERATION_RESULT_UNKNOWN")
	}
	if err != nil {
		return nil, err
	}
	s.after(completed)
	return out, nil
}
func (s Service) CloseEdition(ctx context.Context, edition string) (any, error) {
	id, err := uid(edition)
	if err != nil {
		return nil, err
	}
	err = s.UoW.Within(ctx, Scope{Edition: id}, func(tx Session) error {
		edition, err := tx.Participation.LockEditionForClose()
		if err != nil {
			return err
		}
		if err = edition.CheckEnded(); err != nil {
			return err
		}
		return tx.Participation.MarkEditionEnded()
	})
	if err != nil {
		return nil, err
	}
	return map[string]string{"status": "ENDED"}, nil
}
func (s Service) Read(ctx context.Context, user, edition, kind, before, status string) (any, error) {
	return s.Queries.Read(ctx, user, edition, kind, before, status)
}
func (s Service) Internal(ctx context.Context, action, edition, user, after string) (any, error) {
	if action == "close" {
		return s.CloseEdition(ctx, edition)
	}
	return s.Queries.Internal(ctx, action, edition, user, after)
}
func (s Service) InitializeAccount(ctx context.Context, c AccountCommand) error {
	user, err := uid(c.User)
	if err != nil {
		return fault.New("INVALID_EVENT_USER")
	}
	event, err := uid(c.EventID)
	if err != nil {
		return fault.New("INVALID_EVENT")
	}
	return s.UoW.Within(ctx, Scope{User: user}, func(tx Session) error { return (assetsapp.Service{Repo: tx.Assets}).InitializeAccount(event) })
}
func (s Service) GrantRankingAward(ctx context.Context, c RankingAwardCommand) error {
	user, err := uid(c.User)
	if err != nil {
		return fault.New("INVALID_AWARD")
	}
	edition, err := uid(c.Edition)
	if err != nil {
		return fault.New("INVALID_AWARD")
	}
	award, err := uid(c.Award)
	if err != nil {
		return fault.New("INVALID_AWARD")
	}
	item, err := uid(c.Item)
	if err != nil || c.Quantity <= 0 || item == 1001 || item == 1002 {
		return fault.New("INVALID_AWARD")
	}
	return s.UoW.Within(ctx, Scope{User: user, Edition: edition}, func(tx Session) error {
		// Wallet serializes manual selection, settlement and ranking reward grants.
		if err := tx.Assets.LockWallet(); err != nil {
			return err
		}
		grant, err := tx.Assets.FindRankingGrant(award)
		if errors.Is(err, fault.ErrNotFound) {
			grant, err = s.Runtime.NewID(ctx)
			if err != nil {
				return err
			}
			err = tx.Assets.InsertRankingGrant(grant, award, c.Item, c.Name, item, c.Quantity)
		}
		if err != nil {
			return err
		}
		eventID, err := s.Runtime.NewID(ctx)
		if err != nil {
			return err
		}
		ev := Event{ID: fmt.Sprint(eventID), Type: "RankingAwardGranted.v1", Payload: encode(map[string]string{"award_id": c.Award, "user_id": c.User, "edition_id": c.Edition, "grant_id": fmt.Sprint(grant)})}
		return tx.Outbox.Append(ev, user, award)
	})
}
func (s Service) Bootstrap(ctx context.Context) error {
	candidates, err := s.Queries.BootstrapCandidates(ctx)
	if err != nil {
		return err
	}
	for _, c := range candidates {
		if _, err = s.Join(ctx, c); err != nil {
			return err
		}
	}
	return nil
}
