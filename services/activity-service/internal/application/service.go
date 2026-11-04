package application

import (
	"activity/pkg/platform"
	"activity/services/activity-service/internal/appeffects"
	"activity/services/activity-service/internal/appstate"
	assetsapp "activity/services/activity-service/internal/assets/application"
	lotteryapp "activity/services/activity-service/internal/lottery/application"
	participationapp "activity/services/activity-service/internal/participation/application"
	"activity/services/activity-service/internal/uow"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// Legacy exposes the remaining read-only queries and internal manifest/facts
// endpoints. All mutating use cases are owned by context application services.
type Legacy interface {
	Read(context.Context, string, string, string, string, string) (any, error)
	Internal(context.Context, string, string, string, string) (any, error)
}

type UserLock interface {
	Acquire(context.Context, string) (func(), error)
}

type Service struct {
	UoW     uow.UnitOfWork
	Lock    UserLock
	Effects appeffects.Effects
	Legacy  Legacy
}

var operationPattern = regexp.MustCompile(`^[A-Za-z0-9_:-]{1,64}$`)

func (s Service) Execute(ctx context.Context, user string, c appstate.Command) (any, error) {
	if _, e := platform.UID(c.Edition); e != nil {
		return nil, e
	}
	if (c.Action == "draw" || c.Action == "purchase") && !operationPattern.MatchString(c.Operation) {
		return nil, platform.Fail(400, "INVALID_OPERATION_ID")
	}
	if c.Action == "draw" && (c.Count != 1 && c.Count != 10 && c.Count != 30) {
		return nil, platform.Fail(400, "INVALID_DRAW_COUNT")
	}
	switch c.Action {
	case "join":
		return s.mutate(ctx, user, c, true, func(st *appstate.State) (any, error) {
			return st.Summary(), nil
		})
	case "settle":
		return s.mutate(ctx, user, c, false, func(st *appstate.State) (any, error) {
			if e := participationapp.SettleUser(st); e != nil {
				return nil, e
			}
			return map[string]string{"status": "COMPLETED"}, nil
		})
	case "purchase":
		return s.purchase(ctx, user, c)
	case "draw":
		return s.draw(ctx, user, c)
	case "select":
		return s.selectReward(ctx, user, c)
	case "stage":
		return s.mutate(ctx, user, c, true, func(st *appstate.State) (any, error) {
			return participationapp.ClaimStage(st, c.Stage, "USER")
		})
	case "chest-open":
		return s.mutate(ctx, user, c, true, func(st *appstate.State) (any, error) {
			return participationapp.OpenChest(st)
		})
	case "chest-confirm":
		return s.mutate(ctx, user, c, false, func(st *appstate.State) (any, error) {
			return participationapp.ConfirmChest(st, c.ID)
		})
	case "next-round":
		return s.mutate(ctx, user, c, true, func(st *appstate.State) (any, error) {
			return participationapp.NextRound(st, c.Round)
		})
	case "preview":
		return s.mutate(ctx, user, c, true, func(st *appstate.State) (any, error) {
			return lotteryapp.Preview(st)
		})
	case "preview-confirm":
		return s.mutate(ctx, user, c, true, func(st *appstate.State) (any, error) {
			return lotteryapp.ConfirmPreview(st)
		})
	case "preview-abandon":
		return s.mutate(ctx, user, c, false, func(st *appstate.State) (any, error) {
			return lotteryapp.AbandonPreview(st)
		})
	default:
		return nil, platform.Fail(400, "UNKNOWN_ACTION")
	}
}

func (s Service) Read(ctx context.Context, user, edition, kind, before, status string) (any, error) {
	if s.Legacy == nil {
		return nil, platform.Fail(503, "DEPENDENCY_UNAVAILABLE")
	}
	return s.Legacy.Read(ctx, user, edition, kind, before, status)
}

func (s Service) Internal(ctx context.Context, action, edition, user, after string) (any, error) {
	if action == "close" {
		id, e := platform.UID(edition)
		if e != nil {
			return nil, e
		}
		e = s.UoW.Within(ctx, func(tx uow.Tx) error {
			st := &appstate.State{Tx: tx, Ctx: ctx, Edition: id}
			return st.CloseEdition()
		})
		if e != nil {
			return nil, e
		}
		return map[string]string{"status": "ENDED"}, nil
	}
	if s.Legacy == nil {
		return nil, platform.Fail(503, "DEPENDENCY_UNAVAILABLE")
	}
	return s.Legacy.Internal(ctx, action, edition, user, after)
}

func (s Service) InitializeAccount(ctx context.Context, ev platform.Event) error {
	if ev.Type != "AccountCreated.v1" {
		return platform.Fail(409, "INVALID_EVENT_TYPE")
	}
	var payload struct {
		User string `json:"user_id"`
	}
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		return platform.Fail(409, "INVALID_EVENT")
	}
	user, err := platform.UID(payload.User)
	if err != nil {
		return platform.Fail(409, "INVALID_EVENT_USER")
	}
	eventID, err := platform.UID(ev.ID)
	if err != nil {
		return platform.Fail(409, "INVALID_EVENT")
	}
	return s.UoW.Within(ctx, func(tx uow.Tx) error {
		st := &appstate.State{Tx: tx, Ctx: ctx, User: user}
		return st.InitializeAccount(eventID)
	})
}

func (s Service) GrantRankingAward(ctx context.Context, ev platform.Event) error {
	var payload struct {
		Award   string `json:"award_id"`
		User    string `json:"user_id"`
		Edition string `json:"edition_id"`
		Item    string `json:"item_id"`
		Name    string `json:"item_name"`
		Qty     int64  `json:"quantity"`
	}
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		return platform.Fail(409, "INVALID_AWARD")
	}
	user, err := platform.UID(payload.User)
	if err != nil {
		return platform.Fail(409, "INVALID_AWARD")
	}
	edition, err := platform.UID(payload.Edition)
	if err != nil {
		return platform.Fail(409, "INVALID_AWARD")
	}
	award, err := platform.UID(payload.Award)
	if err != nil {
		return platform.Fail(409, "INVALID_AWARD")
	}
	item, err := platform.UID(payload.Item)
	if err != nil || payload.Qty <= 0 || item == 1001 || item == 1002 {
		return platform.Fail(409, "INVALID_AWARD")
	}
	return s.UoW.Within(ctx, func(tx uow.Tx) error {
		st := &appstate.State{Tx: tx, Ctx: ctx, User: user, Edition: edition}
		grant, err := st.FindRankingGrant(award)
		if errors.Is(err, uow.ErrNoRows) {
			if err = st.LockWallet(); err != nil {
				return err
			}
			grant = platform.ID()
			if err = st.InsertRankingGrant(grant, award, payload.Item, payload.Name, item, payload.Qty); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		return st.AddRankingReceipt(award, map[string]string{"award_id": payload.Award, "user_id": payload.User, "edition_id": payload.Edition, "grant_id": fmt.Sprint(grant)})
	})
}

func (s Service) recoverDraw(ctx context.Context, user, edition uint64, c appstate.Command) (any, bool, error) {
	var out any
	var found bool
	err := s.UoW.Read(ctx, func(tx uow.Tx) error {
		st := &appstate.State{Tx: tx, Ctx: ctx, User: user, Edition: edition}
		var e error
		out, found, e = st.Recover(c)
		return e
	})
	return out, found, err
}

func (s Service) draw(ctx context.Context, userID string, c appstate.Command) (any, error) {
	user, e := platform.UID(userID)
	if e != nil {
		return nil, e
	}
	edition, e := platform.UID(c.Edition)
	if e != nil {
		return nil, e
	}
	if out, found, e := s.recoverDraw(ctx, user, edition, c); e != nil {
		return nil, e
	} else if found {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()
	if s.Lock == nil {
		return nil, platform.Fail(503, "DEPENDENCY_UNAVAILABLE")
	}
	release, e := s.Lock.Acquire(ctx, userID)
	if e != nil {
		return nil, e
	}
	defer release()
	var out any
	var projections []platform.Event
	var messages []map[string]any
	e = s.UoW.Within(ctx, func(tx uow.Tx) error {
		st := &appstate.State{Tx: tx, Ctx: ctx, User: user, Edition: edition, Biz: platform.ID(), Nickname: c.DisplayName}
		if recovered, found, e := st.Recover(c); e != nil {
			return e
		} else if found {
			out = recovered
			return nil
		}
		if e := st.Load(true); e != nil {
			return e
		}
		v, e := lotteryapp.Draw(st, c, nil, 0)
		if e != nil {
			return e
		}
		out = v
		projections = st.Projections
		messages = st.Messages
		return nil
	})
	if errors.Is(e, uow.ErrCommitUnknown) {
		return nil, platform.Fail(503, "OPERATION_RESULT_UNKNOWN")
	}
	if e != nil {
		return nil, e
	}
	for _, ev := range projections {
		s.Effects.Project(ev)
	}
	s.Effects.Enqueue(messages)
	return out, nil
}

func (s Service) mutate(ctx context.Context, userID string, c appstate.Command, active bool, fn func(*appstate.State) (any, error)) (any, error) {
	user, e := platform.UID(userID)
	if e != nil {
		return nil, e
	}
	edition, e := platform.UID(c.Edition)
	if e != nil {
		return nil, e
	}
	ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()
	if s.Lock == nil {
		return nil, platform.Fail(503, "DEPENDENCY_UNAVAILABLE")
	}
	release, e := s.Lock.Acquire(ctx, userID)
	if e != nil {
		return nil, e
	}
	defer release()
	var out any
	var projections []platform.Event
	var messages []map[string]any
	e = s.UoW.Within(ctx, func(tx uow.Tx) error {
		st := &appstate.State{Tx: tx, Ctx: ctx, User: user, Edition: edition, Biz: platform.ID(), Nickname: c.DisplayName}
		if e := st.Load(active); e != nil {
			return e
		}
		v, e := fn(st)
		if e != nil {
			return e
		}
		out = v
		projections = st.Projections
		messages = st.Messages
		return nil
	})
	if errors.Is(e, uow.ErrCommitUnknown) {
		return nil, platform.Fail(503, "OPERATION_RESULT_UNKNOWN")
	}
	if e != nil {
		return nil, e
	}
	for _, ev := range projections {
		s.Effects.Project(ev)
	}
	s.Effects.Enqueue(messages)
	return out, nil
}

func (s Service) selectReward(ctx context.Context, userID string, c appstate.Command) (any, error) {
	user, e := platform.UID(userID)
	if e != nil {
		return nil, e
	}
	edition, e := platform.UID(c.Edition)
	if e != nil {
		return nil, e
	}
	ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()
	release, e := s.Lock.Acquire(ctx, userID)
	if e != nil {
		return nil, e
	}
	defer release()
	var out any
	var messages []map[string]any
	e = s.UoW.Within(ctx, func(tx uow.Tx) error {
		st := &appstate.State{Tx: tx, Ctx: ctx, User: user, Edition: edition, Biz: platform.ID()}
		if e := st.Load(false); e != nil {
			return e
		}
		var e error
		out, e = assetsapp.SelectReward(st, c.ID, c.Item)
		if e != nil {
			return e
		}
		messages = st.Messages
		return nil
	})
	if errors.Is(e, uow.ErrCommitUnknown) {
		return nil, platform.Fail(503, "OPERATION_RESULT_UNKNOWN")
	}
	if e != nil {
		return nil, e
	}
	s.Effects.Enqueue(messages)
	return out, nil
}

func (s Service) purchase(ctx context.Context, userID string, c appstate.Command) (any, error) {
	user, e := platform.UID(userID)
	if e != nil {
		return nil, e
	}
	edition, e := platform.UID(c.Edition)
	if e != nil {
		return nil, e
	}
	ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()
	if s.Lock == nil {
		return nil, platform.Fail(503, "DEPENDENCY_UNAVAILABLE")
	}
	release, e := s.Lock.Acquire(ctx, userID)
	if e != nil {
		return nil, e
	}
	defer release()
	var out any
	var projections []platform.Event
	var messages []map[string]any
	e = s.UoW.Within(ctx, func(tx uow.Tx) error {
		st := &appstate.State{Tx: tx, Ctx: ctx, User: user, Edition: edition, Biz: platform.ID(), Nickname: c.DisplayName}
		if e := st.Load(true); e != nil {
			return e
		}
		var e error
		out, e = assetsapp.Purchase(st, c)
		if e != nil {
			return e
		}
		projections = st.Projections
		messages = st.Messages
		return nil
	})
	if errors.Is(e, uow.ErrCommitUnknown) {
		return nil, platform.Fail(503, "OPERATION_RESULT_UNKNOWN")
	}
	if e != nil {
		return nil, e
	}
	for _, ev := range projections {
		s.Effects.Project(ev)
	}
	s.Effects.Enqueue(messages)
	return out, nil
}
