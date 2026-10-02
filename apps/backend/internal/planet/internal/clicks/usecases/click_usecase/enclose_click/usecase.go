// Package enclose_click is the enclose bonus: while a caller holds the charge and
// has it switched on, a click that closes a shape of its own tiles also takes
// every tile inside it, and spends the charge.
//
// Each tile inside follows the home-soil rule, exactly as a click on it would: a
// tile on another country's own ground that wears its flag is cleared, not taken.
// A bonus is never a way around the rule.
package enclose_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

// Enclosures says whether a caller holds an enclose charge, and how big a shape it may close.
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

// Execute closes shapes only with a click the rule accepted and that took a tile:
// a tile already held changes nothing, so it closes nothing, and a native tile
// the click only cleared is not the caller's, so it is no wall.
func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	holder := bonuses.HolderOf(clicks.PayerOf(ctx))

	if !in.Enclose || u.enclosures.Held(holder).Enclosures == 0 {
		return u.implementation.Execute(ctx, in)
	}

	out, err := u.implementation.Execute(ctx, in)
	if err != nil || out.Outcome != clicks.Taken {
		return out, err
	}

	pockets := u.terrain.PocketsClosedBy(in.TileID, in.CountryID, u.enclosures.EnclosureMaxTiles())

	return out, u.annexer.Annex(ctx, cpctx.RateLimitKey(ctx), holder, in, pockets)
}
