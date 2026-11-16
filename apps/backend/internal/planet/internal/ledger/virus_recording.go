package ledger

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Contagion interface {
	Start(ctx context.Context, virus clicks.Virus, most int, next time.Time) (clicks.Virus, []clicks.Impact, error)
	Spread(ctx context.Context, id clicks.VirusID, most int, next time.Time) (clicks.Virus, []clicks.Impact, error)
}

func NewVirusRecording(contagion Contagion, events Storage, clock cptime.Clock) VirusRecording {
	return VirusRecording{contagion: contagion, events: events, clock: clock}
}

// VirusRecording records each use and tick of a virus that changed a tile, as its own player's act: a virus spreads with
// no request, so its caller is the one kept with it.
type VirusRecording struct {
	contagion Contagion
	events    Storage
	clock     cptime.Clock
}

func (r VirusRecording) Start(ctx context.Context, virus clicks.Virus, most int, next time.Time) (clicks.Virus, error) {
	virus, impacts, err := r.contagion.Start(ctx, virus, most, next)
	r.record(virus, impacts)

	return virus, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
}

func (r VirusRecording) Spread(ctx context.Context, id clicks.VirusID, most int, next time.Time) (clicks.Virus, error) {
	virus, impacts, err := r.contagion.Spread(ctx, id, most, next)
	r.record(virus, impacts)

	return virus, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
}

func (r VirusRecording) record(virus clicks.Virus, impacts []clicks.Impact) {
	impacts = changed(impacts)
	kept := virus.TakeSnapshot()
	if len(impacts) == 0 || kept.Scope == "" {
		return
	}

	by := Caller{Scope: kept.Scope, Account: kept.Account.String()}
	at := r.clock.Now()
	r.events.Append(Infecting{Tile: kept.Tile, Scope: by.Scope, Account: by.Account, Country: kept.Flag, At: at, Impacts: impacts})
	for _, impact := range impacts {
		if impact.Fortified != nil {
			r.events.Append(fortifyingOfImpact(by, at, kept.Flag, impact.Fortified))
		}
	}
	for _, impact := range impacts {
		if impact.KnockedOut != nil {
			r.events.Append(knockingOutOfImpact(by, at, impact.KnockedOut))
		}
	}
}
