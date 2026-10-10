package ledger

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Tiles interface {
	Owner(tile uint32) (string, bool)
	Shields(tile uint32) int
	Clear(ctx context.Context, blast clicks.Blast) (clicks.Blast, error)
	Shield(ctx context.Context, tile uint32, country string, most int) error
}

type Claims interface {
	Click(ctx context.Context, tile uint32, flag string) (clicks.Impact, error)
	Claim(ctx context.Context, tile uint32, flag string) (clicks.Impact, error)
}

func NewRecording(tiles Tiles, claims Claims, events Storage, clock cptime.Clock) Recording {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return Recording{tiles: tiles, claims: claims, events: events, clock: clock}
}

type Recording struct {
	tiles  Tiles
	claims Claims
	events Storage
	clock  cptime.Clock
}

func (r Recording) Owner(tile uint32) (string, bool) {
	return r.tiles.Owner(tile)
}

func (r Recording) Shields(tile uint32) int {
	return r.tiles.Shields(tile)
}

func (r Recording) Click(ctx context.Context, tile uint32, flag string) (clicks.Impact, error) {
	impact, err := r.claims.Click(ctx, tile, flag)
	if err != nil {
		return clicks.Impact{}, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
	}

	if impact.Outcome != clicks.Unchanged {
		r.record(ctx, func(by Caller, at time.Time) Event { return clickOf(by, at, flag, impact) })
	}
	r.recordFortified(ctx, flag, []clicks.Impact{impact})

	return impact, nil
}

func (r Recording) Spread(ctx context.Context, tile uint32, flag string, neighbours []uint32) error {
	impacts, err := r.claim(ctx, flag, neighbours)
	r.record(ctx, func(by Caller, at time.Time) Event {
		return Spreading{Tile: tile, Scope: by.Scope, Account: by.Account, Country: flag, At: at, Impacts: impacts}
	})
	r.recordFortified(ctx, flag, impacts)

	return err
}

func (r Recording) Enclose(ctx context.Context, tile uint32, flag string, inside []uint32) error {
	impacts, err := r.claim(ctx, flag, inside)
	r.record(ctx, func(by Caller, at time.Time) Event {
		return Enclosing{Tile: tile, Scope: by.Scope, Account: by.Account, Country: flag, At: at, Impacts: impacts}
	})
	r.recordFortified(ctx, flag, impacts)

	return err
}

func (r Recording) Clear(ctx context.Context, blast clicks.Blast) (clicks.Blast, error) {
	cleared, err := r.tiles.Clear(ctx, blast)
	if err != nil {
		return cleared, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
	}

	r.record(ctx, func(by Caller, at time.Time) Event {
		return Bombing{Scope: by.Scope, Account: by.Account, At: at, Blast: cleared}
	})

	return cleared, nil
}

func (r Recording) Shield(ctx context.Context, tile uint32, country string, most int) error {
	if err := r.tiles.Shield(ctx, tile, country, most); err != nil {
		return err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
	}

	r.record(ctx, func(by Caller, at time.Time) Event {
		return Shielding{
			Tile: tile, Scope: by.Scope, Account: by.Account, Country: country, Shields: r.tiles.Shields(tile), At: at,
		}
	})

	return nil
}

func (r Recording) claim(ctx context.Context, flag string, tiles []uint32) ([]clicks.Impact, error) {
	impacts := make([]clicks.Impact, 0, len(tiles))
	for _, tile := range tiles {
		impact, err := r.claims.Claim(ctx, tile, flag)
		if err != nil {
			return changed(impacts), fmt.Errorf("failed to claim tile %d: %w", tile, err)
		}
		impacts = append(impacts, impact)
	}

	return changed(impacts), nil
}

func (r Recording) recordFortified(ctx context.Context, flag string, impacts []clicks.Impact) {
	for _, impact := range impacts {
		if impact.Fortified != nil {
			r.record(ctx, func(by Caller, at time.Time) Event { return fortifyingOfImpact(by, at, flag, impact.Fortified) })
		}
	}
}

func (r Recording) record(ctx context.Context, event func(by Caller, at time.Time) Event) {
	scope := cpipscope.Of(cpctx.GetSourceIP(ctx))
	if scope == "" {
		return
	}

	r.events.Append(event(Caller{Scope: scope, Account: cpctx.GetAccount(ctx)}, r.clock.Now()))
}

func clickOf(by Caller, at time.Time, flag string, impact clicks.Impact) Event {
	if impact.Outcome == clicks.Shielded {
		return Striking{
			Tile: impact.Tile, Scope: by.Scope, Account: by.Account, Country: flag, Owner: impact.Owner,
			Shields: impact.Shields, At: at,
		}
	}

	return Taking{Tile: impact.Tile, Scope: by.Scope, Account: by.Account, Country: flag, Previous: impact.Owner, At: at}
}
