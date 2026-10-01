// Package allegiance_click counts each accepted click for its flag, which the toll reads.
package allegiance_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

type Allegiances interface {
	Record(payer clicks.Payer, country string)
}

func New(implementation click_usecase.IUseCase, allegiances Allegiances) *UseCase {
	return &UseCase{implementation: implementation, allegiances: allegiances}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	allegiances    Allegiances
}

// Execute counts only a click the rule accepted, so a refused one leaves an unknown country out of the tally.
func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	out, err := u.implementation.Execute(ctx, in)
	if err == nil {
		u.allegiances.Record(clicks.PayerOf(ctx), in.CountryID)
	}

	return out, err
}
