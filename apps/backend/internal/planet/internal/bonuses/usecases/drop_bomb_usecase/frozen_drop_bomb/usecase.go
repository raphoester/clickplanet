package frozen_drop_bomb

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

type UseCase interface {
	Execute(ctx context.Context, in drop_bomb_usecase.In) (clicks.Blast, error)
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

func (f *Frozen) Execute(ctx context.Context, in drop_bomb_usecase.In) (clicks.Blast, error) {
	if err := f.tempo.Rules().FrozenError(); err != nil {
		return clicks.Blast{}, err //nolint:wrapcheck // the handler maps the sentinel.
	}

	return f.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator passes it on.
}
