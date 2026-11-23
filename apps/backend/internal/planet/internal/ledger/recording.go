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
	Stages
	Owner(tile uint32) (string, bool)
	Shields(tile uint32) int
	Clear(ctx context.Context, blast clicks.Blast) (clicks.Blast, error)
	Shield(ctx context.Context, tile uint32, country string, most int, placement clicks.PlacementID) (int, error)
}

type Claims interface {
	Click(ctx context.Context, tile uint32, flag string) (clicks.Impact, error)
	Claim(ctx context.Context, tile uint32, flag string) (clicks.Impact, error)
	Enclose(ctx context.Context, enclosure clicks.Enclosure, inside []uint32) ([]clicks.Impact, error)
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

// Each write and what it records of it run during one stage (Stages.During): the events name the map and the board
// the write landed on.
func (r Recording) Click(ctx context.Context, tile uint32, flag string) (clicks.Impact, error) {
	var (
		impact clicks.Impact
		err    error
	)
	r.tiles.During(func(on clicks.Stage) {
		impact, err = r.claims.Click(ctx, tile, flag)
		if err != nil {
			return
		}
		if impact.Outcome != clicks.Unchanged {
			r.record(ctx, on, func(by Caller, at time.Time) Event { return clickOf(by, at, flag, impact) })
		}
		r.recordFortified(ctx, on, flag, []clicks.Impact{impact})
		r.recordKnockedOut(ctx, on, []clicks.Impact{impact})
		r.recordCutOff(ctx, on, []clicks.Impact{impact})
	})
	if err != nil {
		return clicks.Impact{}, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
	}

	return impact, nil
}

// Spread answers the traps its takes sprang, the ones before a failure too.
func (r Recording) Spread(ctx context.Context, tile uint32, flag string, neighbours []uint32) ([]clicks.TrapSnapshot, error) {
	var (
		sprung []clicks.TrapSnapshot
		err    error
	)
	r.tiles.During(func(on clicks.Stage) {
		var impacts []clicks.Impact
		impacts, err = r.claim(ctx, flag, neighbours)
		sprung = clicks.Sprung(impacts)
		r.record(ctx, on, func(by Caller, at time.Time) Event {
			return Spreading{Tile: tile, Scope: by.Scope, Account: by.Account, Country: flag, At: at, Impacts: impacts}
		})
		r.recordFortified(ctx, on, flag, impacts)
		r.recordKnockedOut(ctx, on, impacts)
		r.recordCutOff(ctx, on, impacts)
	})

	return sprung, err
}

// Enclose answers the traps its takes sprang, the ones before a failure too.
func (r Recording) Enclose(ctx context.Context, enclosure clicks.Enclosure, inside []uint32) ([]clicks.TrapSnapshot, error) {
	var (
		sprung []clicks.TrapSnapshot
		err    error
	)
	r.tiles.During(func(on clicks.Stage) {
		var impacts []clicks.Impact
		impacts, err = r.claims.Enclose(ctx, enclosure, inside)
		impacts = changed(impacts)
		sprung = clicks.Sprung(impacts)
		r.record(ctx, on, func(by Caller, at time.Time) Event {
			return Enclosing{
				Tile: enclosure.ClosingTile, Scope: by.Scope, Account: by.Account, Country: enclosure.Country, At: at,
				Impacts: impacts,
			}
		})
		r.recordFortified(ctx, on, enclosure.Country, impacts)
		r.recordKnockedOut(ctx, on, impacts)
		r.recordCutOff(ctx, on, impacts)
	})
	if err != nil {
		return sprung, fmt.Errorf("failed to enclose from tile %d: %w", enclosure.ClosingTile, err)
	}

	return sprung, nil
}

func (r Recording) Clear(ctx context.Context, blast clicks.Blast) (clicks.Blast, error) {
	var (
		cleared clicks.Blast
		err     error
	)
	r.tiles.During(func(on clicks.Stage) {
		cleared, err = r.tiles.Clear(ctx, blast)
		if err != nil {
			return
		}
		r.record(ctx, on, func(by Caller, at time.Time) Event {
			return Bombing{Scope: by.Scope, Account: by.Account, At: at, Blast: cleared}
		})
	})
	if err != nil {
		return cleared, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
	}

	return cleared, nil
}

func (r Recording) Shield(
	ctx context.Context, tile uint32, country string, most int, placement clicks.PlacementID,
) (int, error) {
	var (
		shields int
		err     error
	)
	r.tiles.During(func(on clicks.Stage) {
		shields, err = r.tiles.Shield(ctx, tile, country, most, placement)
		if err != nil {
			return
		}
		r.record(ctx, on, func(by Caller, at time.Time) Event {
			return Shielding{Tile: tile, Scope: by.Scope, Account: by.Account, Country: country, Shields: shields, At: at}
		})
	})
	if err != nil {
		return 0, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
	}

	return shields, nil
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

func (r Recording) recordFortified(ctx context.Context, on clicks.Stage, flag string, impacts []clicks.Impact) {
	for _, impact := range impacts {
		if impact.Fortified != nil {
			r.record(ctx, on, func(by Caller, at time.Time) Event { return fortifyingOfImpact(by, at, flag, impact.Fortified) })
		}
	}
}

func (r Recording) recordKnockedOut(ctx context.Context, on clicks.Stage, impacts []clicks.Impact) {
	for _, impact := range impacts {
		if impact.KnockedOut != nil {
			r.record(ctx, on, func(by Caller, at time.Time) Event { return knockingOutOfImpact(by, at, impact.KnockedOut) })
		}
	}
}

func (r Recording) recordCutOff(ctx context.Context, on clicks.Stage, impacts []clicks.Impact) {
	for _, impact := range impacts {
		if impact.CutOff != nil {
			r.record(ctx, on, func(by Caller, at time.Time) Event { return cuttingOffOfImpact(by, at, impact.CutOff) })
		}
	}
}

func (r Recording) record(ctx context.Context, on clicks.Stage, event func(by Caller, at time.Time) Event) {
	scope := cpipscope.Of(cpctx.GetSourceIP(ctx))
	if scope == "" {
		return
	}

	r.events.Append(on, event(Caller{Scope: scope, Account: cpctx.GetAccount(ctx)}, r.clock.Now()))
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
