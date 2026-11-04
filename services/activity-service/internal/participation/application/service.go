package application

import (
	"activity/pkg/platform"
	"context"
	"encoding/json"
	"regexp"
)

type Command struct {
	Edition     string `json:"edition_id" binding:"required"`
	Operation   string `json:"operation_id"`
	Pool        string `json:"pool_code"`
	Count       int    `json:"count"`
	Product     string `json:"product_code"`
	Petals      int64  `json:"petals"`
	Stage       int    `json:"stage"`
	ID          string `json:"id"`
	Item        string `json:"item_id"`
	Round       int    `json:"from_round_no"`
	DisplayName string `json:"-"`
	Action      string `json:"-"`
}
type Repository interface {
	Execute(context.Context, string, Command) (any, error)
	Read(context.Context, string, string, string, string) (any, error)
	Initialize(context.Context, platform.Event) error
	Internal(context.Context, string, string, string, string) (any, error)
}
type Service struct{ Repo Repository }

var operationPattern = regexp.MustCompile(`^[A-Za-z0-9_:-]{1,64}$`)

func (s Service) Execute(ctx context.Context, u string, c Command) (any, error) {
	if _, e := platform.UID(c.Edition); e != nil {
		return nil, e
	}
	if (c.Action == "draw" || c.Action == "purchase") && !operationPattern.MatchString(c.Operation) {
		return nil, platform.Fail(400, "INVALID_OPERATION_ID")
	}
	if c.Action == "draw" && (c.Count != 1 && c.Count != 10 && c.Count != 30) {
		return nil, platform.Fail(400, "INVALID_DRAW_COUNT")
	}
	return s.Repo.Execute(ctx, u, c)
}
func (s Service) Consume(ctx context.Context, e platform.Event) error {
	if !json.Valid(e.Payload) {
		return platform.Fail(409, "INVALID_EVENT")
	}
	return s.Repo.Initialize(ctx, e)
}
