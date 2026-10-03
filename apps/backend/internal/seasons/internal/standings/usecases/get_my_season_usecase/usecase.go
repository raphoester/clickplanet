package get_my_season_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Board interface {
	Place(ctx context.Context, season calendar.Number, account standings.AccountID) (standings.Place, error)
}

type UseCase struct {
	seasons calendar.Calendar
	clock   cptime.Clock
	board   Board
}

func New(seasons calendar.Calendar, clock cptime.Clock, board Board) *UseCase {
	return &UseCase{seasons: seasons, clock: clock, board: board}
}

func (u *UseCase) Execute(ctx context.Context, account standings.AccountID) (standings.Place, error) {
	season, ok := u.seasons.Current(u.clock.Now())
	if !ok {
		return standings.Place{}, nil
	}
	place, err := u.board.Place(ctx, season.Number, account)
	if err != nil {
		return standings.Place{}, fmt.Errorf("failed to read the account's place in season %d: %w", season.Number, err)
	}
	return place, nil
}
