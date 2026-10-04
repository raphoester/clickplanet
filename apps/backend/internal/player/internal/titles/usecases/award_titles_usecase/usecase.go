package award_titles_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Stats interface {
	Stats(ctx context.Context, account players.AccountID) (players.Stats, error)
}

type Accounts interface {
	Account(ctx context.Context, account players.AccountID) (players.Account, error)
}

type Titles interface {
	Unheld(ctx context.Context, account players.AccountID, career titles.Career) (titles.IDs, error)
	Grant(ctx context.Context, account players.AccountID, ids titles.IDs, at time.Time) error
}

type Executor interface {
	Execute(ctx context.Context, account players.AccountID) (titles.IDs, error)
}

type UseCase struct {
	stats    Stats
	accounts Accounts
	titles   Titles
	clock    cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func New(stats Stats, accounts Accounts, titles Titles, clock cptime.Clock) *UseCase {
	return &UseCase{stats: stats, accounts: accounts, titles: titles, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) (titles.IDs, error) {
	stats, err := u.stats.Stats(ctx, account)
	if errors.Is(err, players.ErrNoStats) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the stats: %w", err)
	}

	known, err := u.accounts.Account(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to ask auth about the account: %w", err)
	}

	unheld, err := u.titles.Unheld(ctx, account, titles.CareerOf(stats, known))
	if err != nil {
		return nil, fmt.Errorf("failed to find the titles earned: %w", err)
	}
	if len(unheld) == 0 {
		return nil, nil
	}
	if err := u.titles.Grant(ctx, account, unheld, u.clock.Now()); err != nil {
		return nil, fmt.Errorf("failed to grant the titles: %w", err)
	}
	return unheld, nil
}
