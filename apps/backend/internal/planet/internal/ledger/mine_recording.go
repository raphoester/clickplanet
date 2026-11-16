package ledger

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Minefield interface {
	Lay(ctx context.Context, mine clicks.Mine) (clicks.MapVersion, error)
	Detonate(ctx context.Context, rings int) ([]clicks.Detonation, error)
}

func NewMineRecording(minefield Minefield, events Storage, clock cptime.Clock) MineRecording {
	return MineRecording{minefield: minefield, events: events, clock: clock}
}

// MineRecording records each mine gone off that took a tile, as its own player's act: a mine goes off with no request
// of its player's, so its caller is the one kept with it. A mine laid changes no tile, and is recorded nowhere.
type MineRecording struct {
	minefield Minefield
	events    Storage
	clock     cptime.Clock
}

func (r MineRecording) Lay(ctx context.Context, mine clicks.Mine) (clicks.MapVersion, error) {
	return r.minefield.Lay(ctx, mine) //nolint:wrapcheck // a pure delegation: the storage already named what failed.
}

func (r MineRecording) Detonate(ctx context.Context, rings int) ([]clicks.Detonation, error) {
	detonations, err := r.minefield.Detonate(ctx, rings)
	for _, detonation := range detonations {
		r.record(detonation)
	}

	return detonations, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
}

func (r MineRecording) record(detonation clicks.Detonation) {
	impacts := changed(detonation.Impacts)
	mine := detonation.Mine
	if len(impacts) == 0 || mine.Scope == "" {
		return
	}

	by := Caller{Scope: mine.Scope, Account: mine.Account.String()}
	at := r.clock.Now()
	r.events.Append(Mining{Tile: mine.Tile, Scope: by.Scope, Account: by.Account, Country: mine.Flag, At: at, Impacts: impacts})
	for _, impact := range impacts {
		if impact.Fortified != nil {
			r.events.Append(fortifyingOfImpact(by, at, mine.Flag, impact.Fortified))
		}
	}
	for _, impact := range impacts {
		if impact.KnockedOut != nil {
			r.events.Append(knockingOutOfImpact(by, at, impact.KnockedOut))
		}
	}
}
