package bonus_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

type Presence interface {
	Clicked(click bonuses.Click)
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
		u.presence.Clicked(bonuses.ClickOf(clicks.PayerOf(ctx), out.Limited && out.Budget.Empty()))
	}

	return out, err
}
