package antibot_place_defender

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons/usecases/place_defender_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in place_defender_usecase.In) (bonuses.Held, error)
}

type Bans interface {
	Banned(scope, account string) bool
}

func New(implementation UseCase, bans Bans) *Decorator {
	return &Decorator{implementation: implementation, bans: bans}
}

type Decorator struct {
	implementation UseCase
	bans           Bans
}

// Still spends the defender: one left in hand would tell a banned caller it was refused.
func (d *Decorator) Execute(ctx context.Context, in place_defender_usecase.In) (bonuses.Held, error) {
	payer := clicks.PayerOf(ctx)
	in.Dud = d.bans.Banned(payer.Scope, payer.Account)

	return d.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator adds a flag, not a sentence.
}
