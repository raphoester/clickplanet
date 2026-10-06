package frozen_place_shield

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_shield_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

type UseCase interface {
	Execute(ctx context.Context, in place_shield_usecase.In) (bonuses.Held, error)
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

func (f *Frozen) Execute(ctx context.Context, in place_shield_usecase.In) (bonuses.Held, error) {
	if err := f.tempo.Rules().FrozenError(); err != nil {
		return bonuses.Held{}, err //nolint:wrapcheck // the handler maps the sentinel.
	}

	return f.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator passes it on.
}
