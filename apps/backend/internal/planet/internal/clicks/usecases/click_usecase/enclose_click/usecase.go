package enclose_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

type Enclosures interface {
	Held(ctx context.Context, holder bonuses.Holder) (bonuses.Held, error)
}

func New(implementation click_usecase.IUseCase, enclosures Enclosures, terrain bonuses.Terrain, annexer Annexer, maxTiles int) *UseCase {
	return &UseCase{implementation: implementation, enclosures: enclosures, terrain: terrain, annexer: annexer, maxTiles: maxTiles}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	enclosures     Enclosures
	terrain        bonuses.Terrain
	annexer        Annexer
	maxTiles       int
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	payer := clicks.PayerOf(ctx)
	holder := bonuses.HolderOf(payer)

	if !in.Enclose {
		return u.implementation.Execute(ctx, in)
	}

	held, err := u.enclosures.Held(ctx, holder)
	if err != nil {
		return click_usecase.Out{}, err
	}

	out, err := u.implementation.Execute(ctx, in)
	if err != nil || held.Enclosures == 0 || out.Outcome != clicks.Taken {
		return out, err
	}

	pockets := u.terrain.PocketsClosedBy(in.TileID, in.CountryID, u.maxTiles)

	return out, u.annexer.Annex(ctx, bonuses.EntrantOf(payer), holder, in, pockets)
}
