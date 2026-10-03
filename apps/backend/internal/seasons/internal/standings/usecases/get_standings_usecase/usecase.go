package get_standings_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Board interface {
	Top(ctx context.Context, season calendar.Number, country standings.Country) ([]standings.Standing, error)
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

type UseCase struct {
	seasons   calendar.Calendar
	clock     cptime.Clock
	countries CountryChecker
	board     Board
}

func New(seasons calendar.Calendar, clock cptime.Clock, countries CountryChecker, board Board) *UseCase {
	return &UseCase{seasons: seasons, clock: clock, countries: countries, board: board}
}

func (u *UseCase) Execute(ctx context.Context, country standings.Country) ([]standings.Standing, error) {
	if country != "" && !u.countries.CheckCountry(string(country)) {
		return nil, fmt.Errorf("%w: %q", standings.ErrUnknownCountry, country)
	}
	season, ok := u.seasons.Current(u.clock.Now())
	if !ok {
		return []standings.Standing{}, nil
	}
	top, err := u.board.Top(ctx, season.Number, country)
	if err != nil {
		return nil, fmt.Errorf("failed to read the top of season %d: %w", season.Number, err)
	}
	return top, nil
}
