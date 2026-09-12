//go:build testing

// Disc and Position have no production caller yet — the spread-click bonus is the follow-up, and
// nothing reads a position until a bonus depends on distance. They sit behind the testing tag for
// the reason cpctx.GetSessionID does: `make deadcode` reports production code only a test calls.
// Deleting the tag line is the whole of wiring them up.

package clicks

import (
	"slices"
	"sync"
)

// Position is where tile id sits on the unit sphere.
func (g *Geography) Position(id uint32) (Vec3, bool) {
	if id == 0 || id > g.stats.Tiles {
		return Vec3{}, false
	}

	i := int(id-1) * 3
	return Vec3{
		X: float64(g.positions[i]),
		Y: float64(g.positions[i+1]),
		Z: float64(g.positions[i+2]),
	}, true
}

// Disc returns id and every tile within radius steps of it, ascending — 1+3r(r+1) inland, less
// wherever the land runs out. Computed rather than stored: a radius-3 table would be ~38 MB to
// save ~50 microseconds, behind a 1 click/sec/IP throttle.
func (g *Geography) Disc(id uint32, radius int) []uint32 {
	if id == 0 || id > g.stats.Tiles {
		return nil
	}

	w := takeWalker(g.stats.Tiles)
	defer walkers.Put(w)

	w.begin()
	w.visit(id)

	found := []uint32{id}
	frontier := []uint32{id}

	for range radius {
		frontier = g.nextRing(frontier, w)
		// Everything in reach is already found: an island, or a landmass smaller than the radius.
		if len(frontier) == 0 {
			break
		}
		found = append(found, frontier...)
	}

	slices.Sort(found)

	return found
}

// nextRing returns the tiles one step out from frontier that this search has not reached yet, and
// marks them reached so a later ring does not claim them again.
func (g *Geography) nextRing(frontier []uint32, w *walker) []uint32 {
	var ring []uint32

	for _, tile := range frontier {
		for _, neighbour := range g.Neighbours(tile) {
			if w.visit(neighbour) {
				ring = append(ring, neighbour)
			}
		}
	}

	return ring
}

// walker is the search scratch: clearing 257,948 entries per call would cost more than the
// search, so each pass stamps its own generation instead.
type walker struct {
	seen       []uint32
	generation uint32
}

// One per concurrent caller: the map is shared and immutable, this is not.
var walkers sync.Pool

func takeWalker(tiles uint32) *walker {
	w, _ := walkers.Get().(*walker)
	if w == nil {
		w = &walker{}
	}
	if uint32(len(w.seen)) < tiles+1 {
		w.seen = make([]uint32, tiles+1)
		w.generation = 0
	}
	return w
}

func (w *walker) begin() {
	// Generation 0 means "never seen", so wrapping is the one time the array has to be cleared.
	if w.generation == ^uint32(0) {
		clear(w.seen)
		w.generation = 0
	}
	w.generation++
}

// visit reports whether the tile had not already been reached by this search.
func (w *walker) visit(id uint32) bool {
	if w.seen[id] == w.generation {
		return false
	}
	w.seen[id] = w.generation
	return true
}
