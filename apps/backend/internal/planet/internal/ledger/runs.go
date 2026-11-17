package ledger

import (
	"sort"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Owners interface {
	Owner(tile uint32) (string, bool)
}

// NewRuns follows a caller's runs on one stage, the world's map on the board it plays: a take an event names another
// stage of, an arena's map or another board, neither breaks nor makes one.
func NewRuns(caller Caller, followed clicks.Stage) *Runs {
	return &Runs{caller: caller, followed: followed, touched: cpcolls.NewSet[uint32](), runs: make(map[uint32]run)}
}

type Runs struct {
	caller   Caller
	followed clicks.Stage
	touched  *cpcolls.Set[uint32]
	runs     map[uint32]run
}

type run struct {
	country string
	before  string
}

func (r *Runs) See(on clicks.Stage, event Event) {
	if on == r.followed {
		event.Replay(r.see)
	}
}

func (r *Runs) see(taking Taking) {
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
