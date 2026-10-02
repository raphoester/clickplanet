package get_author_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Codes interface {
	Assign(ctx context.Context, account players.AccountID) error
}

type UseCase struct {
	authors players.AuthorStore
	codes   Codes
	clock   cptime.Clock
}

func New(authors players.AuthorStore, codes Codes, clock cptime.Clock) *UseCase {
	return &UseCase{authors: authors, codes: codes, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) (players.Author, error) {
	author, err := players.AuthorOf(ctx, u.authors, account)
	if errors.Is(err, players.ErrNoGuestCode) {
		if err := u.codes.Assign(ctx, account); err != nil {
			return players.Author{}, fmt.Errorf("failed to give the guest a code: %w", err)
		}
		author, err = players.AuthorOf(ctx, u.authors, account)
	}
	if err != nil {
		return players.Author{}, err //nolint:wrapcheck // AuthorOf says what failed.
	}

	return author.Shown(players.DayOf(u.clock.Now())), nil
}
