package get_season_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type UseCase struct {
	seasons calendar.Calendar
	clock   cptime.Clock
}

func New(seasons calendar.Calendar, clock cptime.Clock) *UseCase {
	return &UseCase{seasons: seasons, clock: clock}
}

// Execute answers the season in play and the next to start, nil for none.
func (u *UseCase) Execute(context.Context) (current, next *calendar.Season) {
	now := u.clock.Now()
	return found(u.seasons.Current(now)), found(u.seasons.Next(now))
}

func found(season calendar.Season, ok bool) *calendar.Season {
	if !ok {
		return nil
	}
	return &season
}
