// Package get_authors_usecase says who each of many accounts is, for another module showing them all at once.
//
// It is the read beside get_author_usecase's write: that one gives a guest its code the first time anybody asks
// about it, which is right when somebody is about to post, and wrong on a path that only reads. An account this
// one cannot name is left out, and the caller decides what to show in its place.
package get_authors_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Authors interface {
	Authors(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]players.Author, error)
}

type UseCase struct {
	authors Authors
}

func New(authors Authors) *UseCase {
	return &UseCase{authors: authors}
}

func (u *UseCase) Execute(
	ctx context.Context,
	accounts []players.AccountID,
) (map[players.AccountID]players.Author, error) {
	found, err := u.authors.Authors(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to read who the accounts are: %w", err)
	}
	return found, nil
}
