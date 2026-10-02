package get_authors_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Authors interface {
	Authors(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]players.Author, error)
}

type UseCase struct {
	authors Authors
	clock   cptime.Clock
}

func New(authors Authors, clock cptime.Clock) *UseCase {
	return &UseCase{authors: authors, clock: clock}
}

func (u *UseCase) Execute(
	ctx context.Context,
	accounts []players.AccountID,
) (map[players.AccountID]players.Author, error) {
	found, err := u.authors.Authors(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to read who the accounts are: %w", err)
	}

	today := players.DayOf(u.clock.Now())
	for account, author := range found {
		found[account] = author.Shown(today)
	}
	return found, nil
}
