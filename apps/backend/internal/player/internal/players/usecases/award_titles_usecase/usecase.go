package award_titles_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Stats interface {
	Stats(ctx context.Context, account players.AccountID) (players.Stats, error)
}

type Titles interface {
	Award(ctx context.Context, account players.AccountID, stats players.Stats, at time.Time) error
}

type UseCase struct {
	stats  Stats
	titles Titles
	clock  cptime.Clock
}

func New(stats Stats, titles Titles, clock cptime.Clock) *UseCase {
	return &UseCase{stats: stats, titles: titles, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) error {
	stats, err := u.stats.Stats(ctx, account)
	if errors.Is(err, players.ErrNoStats) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read the stats: %w", err)
	}

	if err := u.titles.Award(ctx, account, stats, u.clock.Now()); err != nil {
		return fmt.Errorf("failed to award the titles: %w", err)
	}
	return nil
}
