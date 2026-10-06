package enclose_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

type Enclosures interface {
	Held(holder bonuses.Holder) bonuses.Held
	EnclosureMaxTiles() int
}

func New(implementation click_usecase.IUseCase, enclosures Enclosures, terrain bonuses.Terrain, annexer Annexer) *UseCase {
	return &UseCase{implementation: implementation, enclosures: enclosures, terrain: terrain, annexer: annexer}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	enclosures     Enclosures
	terrain        bonuses.Terrain
	annexer        Annexer
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	payer := clicks.PayerOf(ctx)
	holder := bonuses.HolderOf(payer)

	if !in.Enclose || u.enclosures.Held(holder).Enclosures == 0 {
		return u.implementation.Execute(ctx, in)
	}

	out, err := u.implementation.Execute(ctx, in)
	if err != nil || out.Outcome != clicks.Taken {
		return out, err
	}

	pockets := u.terrain.PocketsClosedBy(in.TileID, in.CountryID, u.enclosures.EnclosureMaxTiles())

	return out, u.annexer.Annex(ctx, bonuses.EntrantOf(payer), holder, in, pockets)
}
