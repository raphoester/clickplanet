package get_authors_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Authors interface {
	Authors(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]players.Author, error)
}

type Wardrobe interface {
	WornBy(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]titles.Standing, error)
}

type UseCase struct {
	authors  Authors
	wardrobe Wardrobe
	clock    cptime.Clock
}

func New(authors Authors, wardrobe Wardrobe, clock cptime.Clock) *UseCase {
	return &UseCase{authors: authors, wardrobe: wardrobe, clock: clock}
}

func (u *UseCase) Execute(
	ctx context.Context,
	accounts []players.AccountID,
) (map[players.AccountID]wearing.Author, error) {
	found, err := u.authors.Authors(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to read who the accounts are: %w", err)
	}

	worn, err := u.wardrobe.WornBy(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to read the titles worn: %w", err)
	}

	today := players.DayOf(u.clock.Now())
	shown := make(map[players.AccountID]wearing.Author, len(found))
	for account, author := range found {
		shown[account] = wearing.AuthorOf(author.Shown(today), worn[account])
	}
	return shown, nil
}
