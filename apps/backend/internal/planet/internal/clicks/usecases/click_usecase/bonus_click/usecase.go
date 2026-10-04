package bonus_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

type Presence interface {
	Clicked(entrant bonuses.Entrant, scope string, holder bonuses.Holder)
}

func New(implementation click_usecase.IUseCase, presence Presence) *UseCase {
	return &UseCase{implementation: implementation, presence: presence}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	presence       Presence
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	out, err := u.implementation.Execute(ctx, in)
	if err == nil {
		payer := clicks.PayerOf(ctx)
		u.presence.Clicked(bonuses.EntrantOf(payer), payer.Scope, bonuses.HolderOf(payer))
	}

	return out, err
}
