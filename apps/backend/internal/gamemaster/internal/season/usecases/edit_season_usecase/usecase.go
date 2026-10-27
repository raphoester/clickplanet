package edit_season_usecase

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

// UseCase changes a season that has not ended. Scoring hears of a running one's new end at the next play of the seasons.
type UseCase struct {
	seasons Seasons
	clock   cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, number season.Number, draft season.Draft) error {
	calendar, err := u.seasons.Calendar(ctx)
	if err != nil {
		return fmt.Errorf("could not read the seasons: %w", err)
	}
	edited, err := calendar.Edited(number, draft, u.clock.Now())
	if err != nil {
		return err //nolint:wrapcheck // the calendar says why.
	}
	if err := u.seasons.Save(ctx, edited); err != nil {
		return fmt.Errorf("could not keep season %d: %w", number, err)
	}
	return nil
}
