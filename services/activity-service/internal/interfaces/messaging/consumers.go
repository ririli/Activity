package messaging

import (
	"activity/pkg/platform"
	app "activity/services/activity-service/internal/application"
	"activity/services/activity-service/internal/fault"
	"context"
	"encoding/json"
	"errors"
)

type UseCases interface {
	InitializeAccount(context.Context, app.AccountCommand) error
	GrantRankingAward(context.Context, app.RankingAwardCommand) error
}
type Consumers struct{ App UseCases }

func (c Consumers) InitializeAccount(ctx context.Context, ev platform.Event) (err error) {
	defer func() { err = deliveryError(err) }()
	if ev.Type != "AccountCreated.v1" {
		return fault.New("INVALID_EVENT_TYPE")
	}
	var p struct {
		User string `json:"user_id"`
	}
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return fault.New("INVALID_EVENT")
	}
	return c.App.InitializeAccount(ctx, app.AccountCommand{User: p.User, EventID: ev.ID})
}
func (c Consumers) GrantRankingAward(ctx context.Context, ev platform.Event) (err error) {
	defer func() { err = deliveryError(err) }()
	if ev.Type != "RankingAwardRequested.v1" {
		return fault.New("INVALID_EVENT_TYPE")
	}
	var p struct {
		Award    string `json:"award_id"`
		User     string `json:"user_id"`
		Edition  string `json:"edition_id"`
		Item     string `json:"item_id"`
		Name     string `json:"item_name"`
		Quantity int64  `json:"quantity"`
	}
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return fault.New("INVALID_AWARD")
	}
	return c.App.GrantRankingAward(ctx, app.RankingAwardCommand{User: p.User, Edition: p.Edition, Award: p.Award, Item: p.Item, Name: p.Name, Quantity: p.Quantity})
}

// platform.Consume quarantines permanent failures via platform.Fault. Keep that
// broker-facing contract in this adapter, outside the application.
func deliveryError(err error) error {
	var failure *fault.Error
	if errors.As(err, &failure) {
		return platform.Fail(409, failure.Code)
	}
	return err
}
