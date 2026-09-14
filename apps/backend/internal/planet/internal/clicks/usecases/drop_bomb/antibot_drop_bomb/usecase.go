// Package antibot_drop_bomb is the shadow ban for bombs: a banned caller's bomb is answered OK and clears nothing.
package antibot_drop_bomb

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/drop_bomb"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type UseCase interface {
	Execute(ctx context.Context, in drop_bomb.In) (clicks.Blast, error)
}

type Bans interface {
	Banned(scope string) bool
}

func New(implementation UseCase, bans Bans) *Decorator {
	return &Decorator{implementation: implementation, bans: bans}
}

type Decorator struct {
	implementation UseCase
	bans           Bans
}

// Execute still runs the drop, so the bomb is spent: a bomb that stayed in hand would tell the caller it was refused.
func (d *Decorator) Execute(ctx context.Context, in drop_bomb.In) (clicks.Blast, error) {
	in.Dud = d.bans.Banned(cpctx.RateLimitKey(ctx))

	return d.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator adds a flag, not a sentence.
}
