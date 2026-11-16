package application

import (
	"activity/pkg/hourly"
	assets "activity/services/activity-service/internal/assets/application"
	domain "activity/services/activity-service/internal/assets/domain"
	"activity/services/activity-service/internal/fault"
	"context"
	"errors"
	"strconv"
	"time"
)

type HourlyQueries interface {
	HourlyEditions(context.Context) ([]hourly.Edition, error)
	SealHourly(context.Context, hourly.Window) (hourly.Watermark, error)
	ReplayHourly(context.Context, hourly.Replay) ([]Event, error)
	HourlyPositions(context.Context, hourly.Positions) ([hourly.Lanes]int64, error)
}

type HourlyGrantStore interface {
	FindHourlyGrant(context.Context, hourly.Grant) (uint64, error)
	RecordHourlyGrant(context.Context, hourly.Grant, uint64) error
}

type HourlyService struct {
	Queries HourlyQueries
	UoW     UnitOfWork
	NewID   func(context.Context) (uint64, error)
}

func (s HourlyService) Editions(ctx context.Context) ([]hourly.Edition, error) {
	return s.Queries.HourlyEditions(ctx)
}

func validWindow(w hourly.Window) bool {
	id, err := uid(w.Edition)
	return err == nil && strconv.FormatUint(id, 10) == w.Edition && w.Cutoff > 0 && w.Cutoff%3600 == 0
}

func (s HourlyService) Seal(ctx context.Context, w hourly.Window) (hourly.Watermark, error) {
	if !validWindow(w) {
		return hourly.Watermark{}, fault.New("INVALID_HOURLY_WINDOW")
	}
	return s.Queries.SealHourly(ctx, w)
}

func (s HourlyService) Replay(ctx context.Context, r hourly.Replay) ([]Event, error) {
	if !validWindow(r.Window) || r.Shard < 0 || r.Shard >= hourly.Shards || r.Lane < 0 || r.Lane >= hourly.Lanes || r.After < 0 || r.Through <= r.After {
		return nil, fault.New("INVALID_HOURLY_REPLAY")
	}
	return s.Queries.ReplayHourly(ctx, r)
}

func (s HourlyService) Positions(ctx context.Context, p hourly.Positions) ([hourly.Lanes]int64, error) {
	if !validWindow(p.Window) || p.Shard < 0 || p.Shard >= hourly.Shards {
		return [hourly.Lanes]int64{}, fault.New("INVALID_HOURLY_POSITION")
	}
	return s.Queries.HourlyPositions(ctx, p)
}

func (s HourlyService) Grant(ctx context.Context, g hourly.Grant) (hourly.Receipt, error) {
	var receipt hourly.Receipt
	settlement, se := uid(g.Settlement)
	user, ue := uid(g.User)
	edition, _ := uid(g.Edition)
	if !validWindow(g.Window) || se != nil || ue != nil || strconv.FormatUint(settlement, 10) != g.Settlement ||
		strconv.FormatUint(user, 10) != g.User || g.Rank < 1 || g.Rank > hourly.Winners {
		return receipt, fault.New("INVALID_HOURLY_GRANT")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err := s.UoW.Within(ctx, Scope{User: user, Edition: edition, Biz: settlement}, func(tx Session) error {
		if err := tx.Assets.LockWallet(); err != nil {
			return err
		}
		id, err := tx.HourlyGrant.FindHourlyGrant(ctx, g)
		if err == nil {
			receipt.GrantID = strconv.FormatUint(id, 10)
			return nil
		}
		if !errors.Is(err, fault.ErrNotFound) {
			return err
		}
		wallet, err := tx.Assets.LoadWallet()
		if err != nil {
			return err
		}
		asset := assets.Service{Repo: tx.Assets}
		id, err = s.NewID(ctx)
		if err != nil {
			return err
		}
		err = tx.HourlyGrant.RecordHourlyGrant(ctx, g, id)
		if err != nil {
			return err
		}
		if err := asset.Apply(&wallet, domain.Petal, hourly.Petals, "HOURLY_RANKING_AWARD", "HOURLY_RANKING"); err != nil {
			return err
		}
		if err := tx.Assets.SaveGrants([]assets.Grant{{ID: id, Source: "HOURLY_RANKING", SourceID: settlement, No: 1,
			Prize: domain.Prize{ID: 1002, Code: "1002", Name: "花瓣", Quantity: hourly.Petals}}}); err != nil {
			return err
		}
		receipt.GrantID = strconv.FormatUint(id, 10)
		return nil
	})
	if errors.Is(err, fault.ErrCommitUnknown) {
		return hourly.Receipt{}, fault.New("OPERATION_RESULT_UNKNOWN")
	}
	return receipt, err
}
