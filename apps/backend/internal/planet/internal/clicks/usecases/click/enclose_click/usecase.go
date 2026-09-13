// Package enclose_click is the enclose bonus: while it runs, a click that closes
// a shape of the caller's own tiles also takes every tile inside it.
package enclose_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type Enclosures interface {
	Running(scope string) (*bonus.Enclosure, bool)
}

func New(implementation click.IUseCase, enclosures Enclosures, terrain Terrain, annexer Annexer) *UseCase {
	return &UseCase{implementation: implementation, enclosures: enclosures, terrain: terrain, annexer: annexer}
}

type UseCase struct {
	implementation click.IUseCase
	enclosures     Enclosures
	terrain        Terrain
	annexer        Annexer
}

// Execute closes shapes only with a click the rule accepted and that took a tile:
// a tile already held changes nothing, so it closes nothing.
func (u *UseCase) Execute(ctx context.Context, in click.In) (click.Out, error) {
	scope := cpctx.RateLimitKey(ctx)

	enclosure, running := u.enclosures.Running(scope)
	if !running {
		return u.implementation.Execute(ctx, in)
	}

	alreadyHeld := u.terrain.Holds(in.TileID, in.CountryID)

	out, err := u.implementation.Execute(ctx, in)
	if err != nil || alreadyHeld {
		return out, err
	}

	pockets := u.terrain.PocketsClosedBy(in.TileID, in.CountryID, enclosure.MaxTiles())

	return out, u.annexer.Annex(ctx, scope, in, enclosure, pockets)
}
