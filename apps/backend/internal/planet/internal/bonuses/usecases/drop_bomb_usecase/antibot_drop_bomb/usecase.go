package antibot_drop_bomb

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type UseCase interface {
	Execute(ctx context.Context, in drop_bomb_usecase.In) (clicks.Blast, error)
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

// Still runs the drop: a bomb left in hand would tell a banned caller it was refused.
func (d *Decorator) Execute(ctx context.Context, in drop_bomb_usecase.In) (clicks.Blast, error) {
	payer := clicks.PayerOf(ctx)
	in.Dud = d.bans.Banned(ctx, payer.Scope, payer.Account)

	return d.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator adds a flag, not a sentence.
}
