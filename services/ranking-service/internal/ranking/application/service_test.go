package application

import (
	"context"
	"errors"
	"testing"

	"activity/services/ranking-service/internal/ranking/domain"
)

type fakeBoardRepo struct {
	board domain.Board
	err   error
}

func (f *fakeBoardRepo) Board(context.Context, string, string) (domain.Board, error) {
	return f.board, f.err
}
func (f *fakeBoardRepo) Advance(context.Context, string) (domain.SettlementResult, error) {
	return domain.SettlementResult{}, nil
}

type fakeProfiles struct {
	names map[string]string
	err   error
	ids   []string
}

func (f *fakeProfiles) BatchGet(_ context.Context, ids []string) (map[string]string, error) {
	f.ids = append([]string(nil), ids...)
	return f.names, f.err
}

func fixture() *fakeBoardRepo {
	return &fakeBoardRepo{board: domain.Board{
		Source: "mysql",
		Top:    []domain.Entry{{User: "1", Score: 100, Rank: 1}, {User: "2", Score: 90, Rank: 2}},
		Me:     &domain.Entry{User: "2", Score: 90, Rank: 2},
	}}
}

func TestBoardEnrichesNamesOnce(t *testing.T) {
	repo := fixture()
	profiles := &fakeProfiles{names: map[string]string{"1": "alice", "2": "bob"}}
	s := Service{Repo: repo, Profiles: profiles}
	v, err := s.Board(context.Background(), "1", "2")
	if err != nil {
		t.Fatal(err)
	}
	board := v.(domain.Board)
	if board.Top[0].Nickname != "alice" || board.Me.Nickname != "bob" {
		t.Fatalf("names = %+v", board)
	}
	if len(profiles.ids) != 2 {
		t.Fatalf("expected deduplicated ids, got %v", profiles.ids)
	}
}

func TestBoardProfileFailureKeepsScoresAndPlaceholders(t *testing.T) {
	repo := fixture()
	s := Service{Repo: repo, Profiles: &fakeProfiles{err: errors.New("down")}}
	v, err := s.Board(context.Background(), "1", "2")
	if err != nil {
		t.Fatal(err)
	}
	board := v.(domain.Board)
	if board.Top[0].Score != 100 || board.Top[1].Score != 90 || board.Top[0].Rank != 1 {
		t.Fatalf("scores changed: %+v", board)
	}
	if board.Top[0].Nickname != placeholderName || board.Me.Nickname != placeholderName {
		t.Fatalf("expected placeholders, got %+v", board)
	}
}

func TestBoardProfileAuthFailurePropagates(t *testing.T) {
	repo := fixture()
	s := Service{Repo: repo, Profiles: &fakeProfiles{err: ErrProfileAuth}}
	if _, err := s.Board(context.Background(), "1", "2"); !errors.Is(err, ErrProfileAuth) {
		t.Fatalf("expected ErrProfileAuth, got %v", err)
	}
}

func TestBoardWithoutProfileProviderUsesPlaceholders(t *testing.T) {
	repo := fixture()
	s := Service{Repo: repo}
	v, err := s.Board(context.Background(), "1", "2")
	if err != nil {
		t.Fatal(err)
	}
	board := v.(domain.Board)
	if board.Top[0].Nickname != placeholderName || board.Me.Nickname != placeholderName {
		t.Fatalf("expected placeholders, got %+v", board)
	}
}
