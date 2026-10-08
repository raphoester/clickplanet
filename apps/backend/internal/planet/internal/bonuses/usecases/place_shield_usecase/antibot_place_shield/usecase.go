package antibot_place_shield

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_shield_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type UseCase interface {
	Execute(ctx context.Context, in place_shield_usecase.In) (bonuses.Held, error)
}

type Bans interface {
	Banned(ctx context.Context, scope, account string) bool
}

func New(implementation UseCase, bans Bans) *Decorator {
	return &Decorator{implementation: implementation, bans: bans}
}

type Decorator struct {
	implementation UseCase
	bans           Bans
}

// Still spends the shield: one left in hand would tell a banned caller it was refused.
func (d *Decorator) Execute(ctx context.Context, in place_shield_usecase.In) (bonuses.Held, error) {
	payer := clicks.PayerOf(ctx)
	in.Dud = d.bans.Banned(ctx, payer.Scope, payer.Account)

	return d.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator adds a flag, not a sentence.
}
