package refusal_place_shield

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_shield_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type UseCase interface {
	Execute(ctx context.Context, in place_shield_usecase.In) (bonuses.Held, error)
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

func (d *Decorator) Execute(ctx context.Context, in place_shield_usecase.In) (bonuses.Held, error) {
	if refusal, refused, err := d.refusals.Refusal(ctx, clicks.PayerOf(ctx)); err == nil && refused {
		return bonuses.Held{}, refusal
	}

	return d.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator adds a gate, not a sentence.
}
