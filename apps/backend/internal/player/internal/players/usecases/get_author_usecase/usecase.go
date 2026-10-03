package get_author_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Codes interface {
	Assign(ctx context.Context, account players.AccountID) error
}

type Wardrobe interface {
	Showcase(ctx context.Context, account players.AccountID) (wearing.Showcase, error)
}

type UseCase struct {
	authors  players.AuthorStore
	codes    Codes
	wardrobe Wardrobe
	clock    cptime.Clock
}

func New(authors players.AuthorStore, codes Codes, wardrobe Wardrobe, clock cptime.Clock) *UseCase {
	return &UseCase{authors: authors, codes: codes, wardrobe: wardrobe, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) (wearing.Author, error) {
	author, err := players.AuthorOf(ctx, u.authors, account)
	if errors.Is(err, players.ErrNoGuestCode) {
		if err := u.codes.Assign(ctx, account); err != nil {
			return wearing.Author{}, fmt.Errorf("failed to give the guest a code: %w", err)
		}
		author, err = players.AuthorOf(ctx, u.authors, account)
	}
	if err != nil {
		return wearing.Author{}, err //nolint:wrapcheck // AuthorOf says what failed.
	}

	shown := author.Shown(players.DayOf(u.clock.Now()))
	if shown.Guest {
		return wearing.AuthorOf(shown, titles.Standing{}), nil
	}

	showcase, err := u.wardrobe.Showcase(ctx, account)
	if err != nil {
		return wearing.Author{}, fmt.Errorf("failed to read the title worn: %w", err)
	}
	return wearing.AuthorOf(shown, showcase.Worn), nil
}
