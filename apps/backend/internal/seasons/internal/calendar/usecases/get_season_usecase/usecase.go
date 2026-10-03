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

func (u *UseCase) Execute(context.Context) (calendar.Season, bool) {
	return u.seasons.Current(u.clock.Now())
}
