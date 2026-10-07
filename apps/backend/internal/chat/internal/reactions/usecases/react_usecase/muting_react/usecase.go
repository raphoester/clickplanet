package muting_react

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/usecases/react_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type UseCase interface {
	Execute(ctx context.Context, in react_usecase.In) (react_usecase.Out, error)
}

type Mutes interface {
	MuteError(ctx context.Context, caller mutes.Caller) error
}

func New(implementation UseCase, mutes Mutes) *Decorator {
	return &Decorator{implementation: implementation, mutes: mutes}
}

type Decorator struct {
	implementation UseCase
	mutes          Mutes
}

func (d *Decorator) Execute(ctx context.Context, in react_usecase.In) (react_usecase.Out, error) {
	if err := d.mutes.MuteError(ctx, mutes.NewCaller(in.Account, cpctx.GetSourceIP(ctx))); err != nil {
		return react_usecase.Out{}, err //nolint:wrapcheck // the book named it, and a refusal keeps its type.
	}
	return d.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator adds a gate, not a sentence.
}
