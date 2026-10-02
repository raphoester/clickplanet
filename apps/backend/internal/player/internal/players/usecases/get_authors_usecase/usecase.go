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
