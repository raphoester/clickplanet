package ledger

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// What each tile of an area held at a time: what it held before its first change since, as a replay opens on it.
func NewRewind(area []uint32, at time.Time) *Rewind {
	return &Rewind{area: area, footage: NewFootage(at, at)}
}

type Rewind struct {
	area    []uint32
	footage *Footage
}

func (r *Rewind) See(event Event) {
	r.footage.See(event)
}

// Each tile of the area a change moved off what it held then, put back against the owner it holds now. A change the
// ledger never saw (a reassign, a paint, a revert) is not rewound.
func (r *Rewind) Restorations(owners Owners) []clicks.Restoration {
	var restorations []clicks.Restoration
	for _, tile := range r.area {
		was, changed := r.footage.owners[tile]
		if !changed {
			continue
		}
		if now, _ := owners.Owner(tile); now != was {
			restorations = append(restorations, clicks.Restoration{Tile: tile, From: now, To: was})
		}
	}
	return restorations
}

// How far back the ledger in memory reaches: its retention, from now.
func NewReach(config Config, clock cptime.Clock) Reach {
	return Reach{retention: config.withDefaults().Retention, clock: clock}
}

type Reach struct {
	retention time.Duration
	clock     cptime.Clock
}

func (r Reach) Since() time.Time {
	return r.clock.Now().Add(-r.retention)
}
