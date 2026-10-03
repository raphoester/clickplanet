package get_titles_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Stats interface {
	Stats(ctx context.Context, account players.AccountID) (players.Stats, error)
}

type Titles interface {
	Progress(ctx context.Context, account players.AccountID, career titles.Career) ([]titles.TrackProgress, error)
}

type Wardrobe interface {
	Showcase(ctx context.Context, account players.AccountID) (wearing.Showcase, error)
}

type Dashboard struct {
	Showcase wearing.Showcase
	Tracks   []titles.TrackProgress
}

type UseCase struct {
	stats    Stats
	titles   Titles
	wardrobe Wardrobe
	clock    cptime.Clock
}

func New(stats Stats, titles Titles, wardrobe Wardrobe, clock cptime.Clock) *UseCase {
	return &UseCase{stats: stats, titles: titles, wardrobe: wardrobe, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) (Dashboard, error) {
	stats, err := u.stats.Stats(ctx, account)
	switch {
	case errors.Is(err, players.ErrNoStats):
		stats = players.Stats{Account: account}
	case err != nil:
		return Dashboard{}, fmt.Errorf("failed to read the stats: %w", err)
	}

	career := titles.Career{Stats: stats.AsOf(players.DayOf(u.clock.Now()))}
	tracks, err := u.titles.Progress(ctx, account, career)
	if err != nil {
		return Dashboard{}, fmt.Errorf("failed to read the titles: %w", err)
	}

	showcase, err := u.wardrobe.Showcase(ctx, account)
	if err != nil {
		return Dashboard{}, fmt.Errorf("failed to read the worn title: %w", err)
	}
	return Dashboard{Showcase: showcase, Tracks: tracks}, nil
}
