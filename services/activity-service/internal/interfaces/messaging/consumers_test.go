package messaging

import (
	"activity/pkg/platform"
	app "activity/services/activity-service/internal/application"
	"activity/services/activity-service/internal/fault"
	"context"
	"errors"
	"testing"
)

type appStub struct {
	account app.AccountCommand
	award   app.RankingAwardCommand
	failure error
	calls   int
}

func (s *appStub) InitializeAccount(_ context.Context, c app.AccountCommand) error {
	s.calls++
	s.account = c
	return s.failure
}
func (s *appStub) GrantRankingAward(_ context.Context, c app.RankingAwardCommand) error {
	s.calls++
	s.award = c
	return s.failure
}
func TestConsumersDecodeIntegrationContracts(t *testing.T) {
	s := &appStub{}
	c := Consumers{App: s}
	if err := c.InitializeAccount(context.Background(), platform.Event{ID: "1", Type: "AccountCreated.v1", Payload: []byte(`{"user_id":"7"}`)}); err != nil || s.account != (app.AccountCommand{User: "7", EventID: "1"}) {
		t.Fatalf("account %+v %v", s.account, err)
	}
	if err := c.GrantRankingAward(context.Background(), platform.Event{ID: "2", Type: "RankingAwardRequested.v1", Payload: []byte(`{"award_id":"9","user_id":"7","edition_id":"8","item_id":"3","item_name":"ring","quantity":1}`)}); err != nil || s.award != (app.RankingAwardCommand{User: "7", Edition: "8", Award: "9", Item: "3", Name: "ring", Quantity: 1}) {
		t.Fatalf("award %+v %v", s.award, err)
	}
}
func TestPermanentErrorsRemainQuarantinable(t *testing.T) {
	s := &appStub{failure: fault.New("INVALID_AWARD")}
	c := Consumers{App: s}
	for _, ev := range []platform.Event{{Type: "Wrong", Payload: []byte(`{}`)}, {Type: "RankingAwardRequested.v1", Payload: []byte(`bad`)}, {Type: "RankingAwardRequested.v1", Payload: []byte(`{}`)}} {
		err := c.GrantRankingAward(context.Background(), ev)
		var f *platform.Fault
		if !errors.As(err, &f) || f.Status != 409 {
			t.Fatalf("permanent failure would be retried: %v", err)
		}
	}
	transient := errors.New("database offline")
	s.failure = transient
	err := c.InitializeAccount(context.Background(), platform.Event{Type: "AccountCreated.v1", Payload: []byte(`{"user_id":"7"}`)})
	if !errors.Is(err, transient) {
		t.Fatalf("transient error changed %v", err)
	}
}
