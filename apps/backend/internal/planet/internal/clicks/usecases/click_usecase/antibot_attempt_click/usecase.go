package antibot_attempt_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type AttemptGuard interface {
	Attempted(click antibot.Click)
}

type TilePositions interface {
	Position(tile uint32) (clicks.Vec3, bool)
}

func New(implementation click_usecase.IUseCase, guard AttemptGuard, positions TilePositions, clock cptime.Clock) *UseCase {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &UseCase{implementation: implementation, guard: guard, positions: positions, clock: clock}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	guard          AttemptGuard
	positions      TilePositions
	clock          cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	attempt := antibot.Click{
		Scope:    cpipscope.Of(cpctx.GetSourceIP(ctx)),
		Account:  cpctx.GetAccount(ctx),
		SignedIn: cpctx.GetLinked(ctx),
		Tile:     in.TileID,
		Country:  in.CountryID,
		At:       u.clock.Now(),
	}

	if at, ok := u.positions.Position(in.TileID); ok {
		attempt.Position = antibot.Point{X: at.X, Y: at.Y, Z: at.Z}
	}

	u.guard.Attempted(attempt)

	return u.implementation.Execute(ctx, in)
}
