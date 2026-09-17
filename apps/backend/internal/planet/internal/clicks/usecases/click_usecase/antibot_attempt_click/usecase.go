// Package antibot_attempt_click shows the antibot guard every click tried, outside the throttle.
package antibot_attempt_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// AttemptGuard is the part of antibot.Guard this decorator uses.
type AttemptGuard interface {
	Attempted(click antibot.Click)
}

func New(implementation click_usecase.IUseCase, guard AttemptGuard, clock cptime.Clock) *UseCase {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &UseCase{implementation: implementation, guard: guard, clock: clock}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	guard          AttemptGuard
	clock          cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	u.guard.Attempted(antibot.Click{
		Scope:   cpipscope.Of(cpctx.GetSourceIP(ctx)),
		Account: cpctx.GetAccount(ctx),
		Tile:    in.TileID,
		Country: in.CountryID,
		At:      u.clock.Now(),
	})

	return u.implementation.Execute(ctx, in)
}
