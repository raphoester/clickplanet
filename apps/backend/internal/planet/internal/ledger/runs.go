package ledger

import (
	"sort"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Owners interface {
	Owner(tile uint32) (string, bool)
}

// NewRuns follows a caller's runs on the world's map: a take on an arena's map neither breaks nor makes one, since the
// world's map came back as it was.
func NewRuns(caller Caller, world arenas.Played) *Runs {
	return &Runs{caller: caller, world: world, touched: cpcolls.NewSet[uint32](), runs: make(map[uint32]run)}
}

type Runs struct {
	caller  Caller
	world   arenas.Played
	touched *cpcolls.Set[uint32]
	runs    map[uint32]run
}

type run struct {
	country string
	before  string
}

func (r *Runs) See(event Event) {
	event.Replay(r.see)
}

func (r *Runs) see(taking Taking) {
	if !r.world.Holds(taking.At) {
		return
	}
	current, ours := r.runs[taking.Tile]

	if !r.caller.Made(taking) {
		if ours {
			delete(r.runs, taking.Tile)
		}
		return
	}

	r.touched.Add(taking.Tile)

	before := taking.Previous
	if ours && current.country == taking.Previous {
		before = current.before
	}

	r.runs[taking.Tile] = run{country: taking.Country, before: before}
}

func (r *Runs) Touched() int {
	return r.touched.Len()
}

func (r *Runs) Restorations(owners Owners) []clicks.Restoration {
	restorations := make([]clicks.Restoration, 0, len(r.runs))
	for tile, run := range r.runs {
		if run.before == run.country {
			continue
		}
		if owner, _ := owners.Owner(tile); owner != run.country {
			continue
		}
		restorations = append(restorations, clicks.Restoration{Tile: tile, From: run.country, To: run.before})
	}

	sort.Slice(restorations, func(i, j int) bool { return restorations[i].Tile < restorations[j].Tile })

	return restorations
}
