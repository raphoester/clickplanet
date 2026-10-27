package add_season_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Seasons interface {
	Calendar(ctx context.Context) (season.Calendar, error)
	Save(ctx context.Context, season season.Season) error
}

func New(seasons Seasons, clock cptime.Clock) *UseCase {
	return &UseCase{seasons: seasons, clock: clock}
}

// UseCase schedules a season to come, under the next number.
type UseCase struct {
	seasons Seasons
	clock   cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, draft season.Draft) (season.Number, error) {
	calendar, err := u.seasons.Calendar(ctx)
	if err != nil {
		return 0, fmt.Errorf("could not read the seasons: %w", err)
	}
	added, err := calendar.Added(draft, u.clock.Now())
	if err != nil {
		return 0, err //nolint:wrapcheck // the calendar says why.
	}
	if err := u.seasons.Save(ctx, added); err != nil {
		return 0, fmt.Errorf("could not keep the season: %w", err)
	}
	return added.TakeSnapshot().Number, nil
}
