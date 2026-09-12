// Package bonus_click marks a caller as playing, so boxes go to people who are.
package bonus_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type Presence interface {
	Clicked(scope string)
}

func New(implementation click.IUseCase, presence Presence) *UseCase {
	return &UseCase{implementation: implementation, presence: presence}
}

type UseCase struct {
	implementation click.IUseCase
	presence       Presence
}

// Execute marks presence only for a click that was accepted, so neither a
// refusal nor a click a shadow ban dropped counts as playing.
func (u *UseCase) Execute(ctx context.Context, in click.In) (click.Out, error) {
	out, err := u.implementation.Execute(ctx, in)
	if err == nil {
		u.presence.Clicked(cpctx.RateLimitKey(ctx))
	}

	return out, err
}
