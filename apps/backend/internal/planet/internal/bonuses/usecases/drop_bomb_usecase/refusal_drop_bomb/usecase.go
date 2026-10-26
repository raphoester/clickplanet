package refusal_drop_bomb

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type UseCase interface {
	Execute(ctx context.Context, in drop_bomb_usecase.In) (clicks.Blast, error)
}

type Refusals interface {
	Refusal(ctx context.Context, payer clicks.Payer) (clicks.Refusal, bool, error)
}

func New(implementation UseCase, refusals Refusals) *Decorator {
	return &Decorator{implementation: implementation, refusals: refusals}
}

type Decorator struct {
	implementation UseCase
	refusals       Refusals
}

func (d *Decorator) Execute(ctx context.Context, in drop_bomb_usecase.In) (clicks.Blast, error) {
	if refusal, refused, err := d.refusals.Refusal(ctx, clicks.PayerOf(ctx)); err == nil && refused {
		return clicks.Blast{}, refusal
	}

	return d.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator adds a gate, not a sentence.
}
