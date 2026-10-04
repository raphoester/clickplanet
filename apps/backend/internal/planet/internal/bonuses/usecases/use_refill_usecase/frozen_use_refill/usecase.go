package frozen_use_refill

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/use_refill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

type UseCase interface {
	Execute(ctx context.Context, in use_refill_usecase.In) (use_refill_usecase.Out, error)
}

type Tempo interface {
	Rules() tempo.Rules
}

func New(implementation UseCase, tempo Tempo) *Frozen {
	return &Frozen{implementation: implementation, tempo: tempo}
}

type Frozen struct {
	implementation UseCase
	tempo          Tempo
}

func (f *Frozen) Execute(ctx context.Context, in use_refill_usecase.In) (use_refill_usecase.Out, error) {
	if err := f.tempo.Rules().FrozenError(); err != nil {
		return use_refill_usecase.Out{}, err //nolint:wrapcheck // the handler maps the sentinel.
	}

	return f.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator passes it on.
}
