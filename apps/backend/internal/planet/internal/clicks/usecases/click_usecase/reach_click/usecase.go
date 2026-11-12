package reach_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

type Reach interface {
	Refuses(flag string, tile uint32, rings int) error
}

func New(implementation click_usecase.IUseCase, reach Reach) *UseCase {
	return &UseCase{implementation: implementation, reach: reach}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	reach          Reach
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	if err := u.reach.Refuses(in.CountryID, in.TileID, in.Rings); err != nil {
		return click_usecase.Out{}, err //nolint:wrapcheck // the reach names the tile and the flag.
	}

	return u.implementation.Execute(ctx, in)
}
