package detonate_mines_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Minefield interface {
	Detonate(ctx context.Context, rings int) ([]clicks.Detonation, error)
}

type Blasting interface {
	Rings() int
}

type Executor interface {
	Execute(ctx context.Context) (int, error)
}

func New(minefield Minefield, blasting Blasting) *UseCase {
	return &UseCase{minefield: minefield, blasting: blasting}
}

type UseCase struct {
	minefield Minefield
	blasting  Blasting
}

var _ Executor = (*UseCase)(nil)

// Execute sets off every mine a take tripped, at the blast's width in force, even while the map takes no play: the take
// that tripped them was played before.
func (u *UseCase) Execute(ctx context.Context) (int, error) {
	detonations, err := u.minefield.Detonate(ctx, u.blasting.Rings())

	return len(detonations), err //nolint:wrapcheck // the minefield names it.
}
