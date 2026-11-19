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

func NewMineRecording(minefield Minefield, stages Stages, events Storage, clock cptime.Clock) MineRecording {
	return MineRecording{minefield: minefield, stages: stages, events: events, clock: clock}
}

// MineRecording records each mine gone off that changed a tile, as its own player's act: a mine goes off with no request
// of its player's, so its caller is the one kept with it. A mine laid changes no tile, and is recorded nowhere.
type MineRecording struct {
	minefield Minefield
	stages    Stages
	events    Storage
	clock     cptime.Clock
}

func (r MineRecording) Lay(ctx context.Context, mine clicks.Mine) (clicks.MapVersion, error) {
	return r.minefield.Lay(ctx, mine) //nolint:wrapcheck // a pure delegation: the storage already named what failed.
}

// Detonate records every blast, then the fortifies, then the knock-outs: the order the explosion wrote them in.
func (r MineRecording) Detonate(ctx context.Context, rings int) ([]clicks.Detonation, error) {
	var (
		detonations []clicks.Detonation
		err         error
	)
	r.stages.During(func(on clicks.Stage) {
		detonations, err = r.minefield.Detonate(ctx, rings)
		r.record(on, detonations)
	})

	return detonations, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
}

func (r MineRecording) record(on clicks.Stage, detonations []clicks.Detonation) {
	at := r.clock.Now()
	var recorded []clicks.Detonation
	for _, detonation := range detonations {
		detonation.Impacts = changed(detonation.Impacts)
		mine := detonation.Mine
		if len(detonation.Impacts) == 0 || mine.Scope == "" {
			continue
		}
		recorded = append(recorded, detonation)
		r.events.Append(on, Mining{
			Tile: mine.Tile, Scope: mine.Scope, Account: mine.Account.String(), Country: mine.Flag, At: at,
			Impacts: detonation.Impacts,
		})
	}
	for _, detonation := range recorded {
		for _, impact := range detonation.Impacts {
			if impact.Fortified != nil {
				r.events.Append(on, fortifyingOfImpact(callerOf(detonation.Mine), at, detonation.Mine.Flag, impact.Fortified))
			}
		}
	}
	for _, detonation := range recorded {
		for _, impact := range detonation.Impacts {
			if impact.KnockedOut != nil {
				r.events.Append(on, knockingOutOfImpact(callerOf(detonation.Mine), at, impact.KnockedOut))
			}
		}
	}
	for _, detonation := range recorded {
		for _, impact := range detonation.Impacts {
			if impact.CutOff != nil {
				r.events.Append(on, cuttingOffOfImpact(callerOf(detonation.Mine), at, impact.CutOff))
			}
		}
	}
}

func callerOf(mine clicks.MineSnapshot) Caller {
	return Caller{Scope: mine.Scope, Account: mine.Account.String()}
}
