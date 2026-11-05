package keep_day_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
)

type Days interface {
	KeepDay(ctx context.Context, day days.Day) error
}

func New(days Days) *UseCase {
	return &UseCase{days: days}
}

// UseCase keeps each day the gamemaster told, so the facts of its time count in its missions.
type UseCase struct {
	days Days
}

func (u *UseCase) Execute(ctx context.Context, day days.Day) error {
	return u.days.KeepDay(ctx, day) //nolint:wrapcheck // the store names the day.
}
