package allegiance_click

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

type Allegiances interface {
	PlayError(ctx context.Context, payer clicks.Payer, flag string) error
}

func New(implementation click_usecase.IUseCase, allegiances Allegiances) *UseCase {
	return &UseCase{implementation: implementation, allegiances: allegiances}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	allegiances    Allegiances
}

// A flag that cannot be read refuses nothing, as a refusal that cannot be read.
func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	if err := u.allegiances.PlayError(ctx, clicks.PayerOf(ctx), in.CountryID); err != nil && !errors.Is(err, allegiances.ErrUnread) {
		return click_usecase.Out{}, err
	}

	return u.implementation.Execute(ctx, in)
}
