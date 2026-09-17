package ledger

import (
	"sort"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Owners interface {
	Owner(tile uint32) (string, bool)
}

func NewRuns(caller Caller) *Runs {
	return &Runs{caller: caller, touched: cpcolls.NewSet[uint32](), runs: make(map[uint32]run)}
}

// Runs is what a revert of one caller gives back, by the rule in the package doc.
type Runs struct {
	caller  Caller
	touched *cpcolls.Set[uint32]
	runs    map[uint32]run
}

type run struct {
	country string
	before  string
}

func (r *Runs) See(taking Taking) {
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

// Touched is how many tiles the caller took, held or not.
func (r *Runs) Touched() int {
	return r.touched.Len()
}

// Restorations gives back every tile the caller still holds, in tile order.
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
