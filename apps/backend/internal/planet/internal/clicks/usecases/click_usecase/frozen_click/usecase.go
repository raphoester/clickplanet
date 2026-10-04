package frozen_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

type Tempo interface {
	Rules() tempo.Rules
}

func New(implementation click_usecase.IUseCase, tempo Tempo) *UseCase {
	return &UseCase{implementation: implementation, tempo: tempo}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	tempo          Tempo
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	if err := u.tempo.Rules().FrozenError(); err != nil {
		return click_usecase.Out{}, err //nolint:wrapcheck // the handler maps the sentinel.
	}

	return u.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator passes it on.
}
