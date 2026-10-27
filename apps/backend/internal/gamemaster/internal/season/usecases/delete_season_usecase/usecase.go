package delete_season_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Seasons interface {
	Calendar(ctx context.Context) (season.Calendar, error)
	Delete(ctx context.Context, number season.Number) error
}

func New(seasons Seasons, clock cptime.Clock) *UseCase {
	return &UseCase{seasons: seasons, clock: clock}
}

// UseCase deletes the last season, when it has not started: scoring never heard of it.
type UseCase struct {
	seasons Seasons
	clock   cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, number season.Number) error {
	calendar, err := u.seasons.Calendar(ctx)
	if err != nil {
		return fmt.Errorf("could not read the seasons: %w", err)
	}
	if err := calendar.Deletable(number, u.clock.Now()); err != nil {
		return err //nolint:wrapcheck // the calendar says why.
	}
	if err := u.seasons.Delete(ctx, number); err != nil {
		return fmt.Errorf("could not delete season %d: %w", number, err)
	}
	return nil
}
