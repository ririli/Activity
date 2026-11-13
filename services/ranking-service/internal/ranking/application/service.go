package application

import (
	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/domain"
	"context"
	"errors"
	"log/slog"
)

var ErrProfileAuth = errors.New("profile authentication failed")

// ProfileProvider resolves public display names. Implementations may cache;
// failures other than ErrProfileAuth must be reflected as placeholder names,
// never as a change to score or rank.
type ProfileProvider interface {
	BatchGet(ctx context.Context, userIDs []string) (map[string]string, error)
}

type Repository interface {
	Board(context.Context, string, string) (domain.Board, error)
	Advance(context.Context, string) (domain.SettlementResult, error)
}
type Service struct {
	Repo     Repository
	Profiles ProfileProvider
}

const placeholderName = "用户"

func (s Service) Advance(ctx context.Context, edition string) (domain.SettlementResult, error) {
	if _, e := platform.UID(edition); e != nil {
		return domain.SettlementResult{}, e
	}
	return s.Repo.Advance(ctx, edition)
}

func (s Service) Board(ctx context.Context, edition, user string) (any, error) {
	if _, e := platform.UID(edition); e != nil {
		return nil, e
	}
	b, e := s.Repo.Board(ctx, edition, user)
	if e != nil {
		return nil, e
	}
	if b.Top == nil {
		b.Top = []domain.Entry{}
	}
	ids := make([]string, 0, len(b.Top)+1)
	seen := map[string]struct{}{}
	add := func(id string) {
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, entry := range b.Top {
		add(entry.User)
	}
	if b.Me != nil {
		add(b.Me.User)
	}
	names := map[string]string{}
	if s.Profiles != nil && len(ids) > 0 {
		got, err := s.Profiles.BatchGet(ctx, ids)
		if err != nil {
			if errors.Is(err, ErrProfileAuth) {
				return nil, err
			}
			slog.Warn("profile lookup failed; using placeholders", "error", err)
		}
		if got != nil {
			names = got
		}
	}
	display := func(id string) string {
		if name := names[id]; name != "" {
			return name
		}
		return placeholderName
	}
	for i := range b.Top {
		b.Top[i].Nickname = display(b.Top[i].User)
	}
	if b.Me != nil {
		b.Me.Nickname = display(b.Me.User)
	}
	return b, nil
}
