// Package move_visit_usecase moves a browser's visit to the account it signed in to, so the roster shows the
// player at once, under its name, and never beside the guest it was.
package move_visit_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

// Authors says who an account is, and gives a guest its code the first time: get_author_usecase.
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

// Execute moves the visit of from to to. A sign-in that keeps the browser on its account changes nothing:
// a guest that links its first identity has no username yet, and a signed-in account keeps the one it had.
// Reading the profile then could only race SetName, and put back the name it replaced.
func (u *UseCase) Execute(ctx context.Context, from, to players.AccountID) error {
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
