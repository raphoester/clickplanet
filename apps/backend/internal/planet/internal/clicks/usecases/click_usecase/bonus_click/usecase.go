package bonus_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

// Presence is the offers' feed: every click the throttle let through and the chain inside it took, struck or left
// unchanged, and only those (ClickTried's TAKEN, STRUCK and UNCHANGED). A refused, throttled, invalid or failed try is
// never heard.
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
		payer := clicks.PayerOf(ctx)
		u.presence.Clicked(bonuses.Click{
			Caller: bonuses.Caller{Scope: payer.Scope, Account: bonuses.Account(payer.Account), Linked: payer.Linked},
			Born:   payer.Created,
			Dry:    out.Limited && out.Budget.Empty(),
		})
	}

	return out, err
}
