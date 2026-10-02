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
	CreatedAt(ctx context.Context, account players.AccountID) (time.Time, error)
}

type Titles interface {
	Award(ctx context.Context, account players.AccountID, career titles.Career, at time.Time) error
}

type UseCase struct {
	stats    Stats
	accounts Accounts
	titles   Titles
	clock    cptime.Clock
}

func New(stats Stats, accounts Accounts, titles Titles, clock cptime.Clock) *UseCase {
	return &UseCase{stats: stats, accounts: accounts, titles: titles, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) error {
	stats, err := u.stats.Stats(ctx, account)
	if errors.Is(err, players.ErrNoStats) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read the stats: %w", err)
	}

	createdAt, err := u.accounts.CreatedAt(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to ask when the account was made: %w", err)
	}

	career := titles.Career{Stats: stats, CreatedAt: createdAt}
	if err := u.titles.Award(ctx, account, career, u.clock.Now()); err != nil {
		return fmt.Errorf("failed to award the titles: %w", err)
	}
	return nil
}
