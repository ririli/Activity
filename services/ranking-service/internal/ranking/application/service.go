package application

import (
	"activity/pkg/platform"
	"context"
)

type Repository interface {
	Consume(context.Context, platform.Event) error
	Board(context.Context, string, string) (any, error)
	Settle(context.Context, string) error
}
type Service struct{ Repo Repository }

func (s Service) Board(ctx context.Context, edition, user string) (any, error) {
	if _, e := platform.UID(edition); e != nil {
		return nil, e
	}
	return s.Repo.Board(ctx, edition, user)
}
