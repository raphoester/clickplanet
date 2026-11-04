package move_visit_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Authors interface {
	Execute(ctx context.Context, account players.AccountID) (players.Author, error)
}

type Visits interface {
	Move(from, to players.AccountID, author players.Author)
}

type UseCase struct {
	authors Authors
	visits  Visits
}

func New(authors Authors, visits Visits) *UseCase {
	return &UseCase{authors: authors, visits: visits}
}

func (u *UseCase) Execute(ctx context.Context, from, to players.AccountID) error {
	// Same account: re-reading its name here could race SetName and restore the old one.
	if from == to {
		return nil
	}

	author, err := u.authors.Execute(ctx, to)
	if err != nil {
		return fmt.Errorf("failed to name the account: %w", err)
	}

	u.visits.Move(from, to, author)
	return nil
}
