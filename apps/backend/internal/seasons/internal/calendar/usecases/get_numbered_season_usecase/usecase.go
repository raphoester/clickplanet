package get_numbered_season_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

type UseCase struct {
	seasons calendar.Calendar
}

func New(seasons calendar.Calendar) *UseCase {
	return &UseCase{seasons: seasons}
}

func (u *UseCase) Execute(_ context.Context, number calendar.Number) (calendar.Season, bool) {
	return u.seasons.Season(number)
}
