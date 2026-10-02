// Package bonus_click marks a caller as playing, so boxes go to people who are.
package bonus_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

// Presence is told who clicked: the scope the boxes are scheduled by, and the holder whose charges keep a
// kind from being offered to it.
type Presence interface {
	Clicked(scope string, holder bonuses.Holder)
}

func New(implementation click_usecase.IUseCase, presence Presence) *UseCase {
	return &UseCase{implementation: implementation, presence: presence}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	presence       Presence
}

// Execute marks presence only for a click that was accepted, so neither a
// refusal nor a click a shadow ban dropped counts as playing.
func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	out, err := u.implementation.Execute(ctx, in)
	if err == nil {
		payer := clicks.PayerOf(ctx)
		u.presence.Clicked(payer.Scope, bonuses.HolderOf(payer))
	}

	return out, err
}
