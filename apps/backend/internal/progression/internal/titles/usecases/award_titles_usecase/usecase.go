package award_titles_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Stats interface {
	Stats(ctx context.Context, account accounts.AccountID) (stats.Stats, error)
}

type Careers interface {
	Of(ctx context.Context, stats stats.Stats) (titles.Career, error)
}

type Titles interface {
	Unheld(ctx context.Context, account accounts.AccountID, career titles.Career) (titles.IDs, error)
	Grant(ctx context.Context, account accounts.AccountID, ids titles.IDs, at time.Time) error
}

type Executor interface {
	Execute(ctx context.Context, account accounts.AccountID) (titles.IDs, error)
}

type UseCase struct {
	stats   Stats
	careers Careers
	titles  Titles
	clock   cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func New(stats Stats, careers Careers, titles Titles, clock cptime.Clock) *UseCase {
	return &UseCase{stats: stats, careers: careers, titles: titles, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, account accounts.AccountID) (titles.IDs, error) {
	counted, err := u.stats.Stats(ctx, account)
	if errors.Is(err, stats.ErrNoStats) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the stats: %w", err)
	}

	career, err := u.careers.Of(ctx, counted)
	if err != nil {
		return nil, fmt.Errorf("failed to read the career: %w", err)
	}

	unheld, err := u.titles.Unheld(ctx, account, career)
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
