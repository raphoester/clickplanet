package get_titles_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Stats interface {
	Stats(ctx context.Context, account players.AccountID) (players.Stats, error)
}

type Titles interface {
	Dashboard(ctx context.Context, account players.AccountID, career titles.Career) (titles.Dashboard, error)
}

type UseCase struct {
	stats  Stats
	titles Titles
	clock  cptime.Clock
}

func New(stats Stats, titles Titles, clock cptime.Clock) *UseCase {
	return &UseCase{stats: stats, titles: titles, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) (titles.Dashboard, error) {
	stats, err := u.stats.Stats(ctx, account)
	switch {
	case errors.Is(err, players.ErrNoStats):
		stats = players.Stats{Account: account}
	case err != nil:
		return titles.Dashboard{}, fmt.Errorf("failed to read the stats: %w", err)
	}

	career := titles.Career{Stats: stats.AsOf(players.DayOf(u.clock.Now()))}
	dashboard, err := u.titles.Dashboard(ctx, account, career)
	if err != nil {
		return titles.Dashboard{}, fmt.Errorf("failed to read the titles: %w", err)
	}
	return dashboard, nil
}
