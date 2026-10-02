package get_author_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Codes interface {
	Assign(ctx context.Context, account players.AccountID) error
}

type UseCase struct {
	authors players.AuthorStore
	codes   Codes
}

func New(authors players.AuthorStore, codes Codes) *UseCase {
	return &UseCase{authors: authors, codes: codes}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) (players.Author, error) {
	author, err := players.AuthorOf(ctx, u.authors, account)
	if !errors.Is(err, players.ErrNoGuestCode) {
		return author, err //nolint:wrapcheck // AuthorOf says what failed.
	}

	if err := u.codes.Assign(ctx, account); err != nil {
		return players.Author{}, fmt.Errorf("failed to give the guest a code: %w", err)
	}
	return players.AuthorOf(ctx, u.authors, account) //nolint:wrapcheck // as above.
}
