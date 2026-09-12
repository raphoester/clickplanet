package clicks

import (
	"math"
	"slices"
	"sync"
)

// Spacing is the mean arc between two touching tiles, in radians.
func (g *Geography) Spacing() float64 {
	total, edges := 0.0, 0

	for id := uint32(1); id <= g.stats.Tiles; id++ {
		from, _ := g.Position(id)
		for _, other := range g.Neighbours(id) {
			to, _ := g.Position(other)
			total += angle(from, to)
			edges++
		}
	}

	if edges == 0 {
		return 0
	}

	return total / float64(edges)
}

// Nearest returns the tile closest to point and the arc to it; a straight scan, once per bomb.
func (g *Geography) Nearest(point Vec3) (uint32, float64) {
	unit, ok := normalize(point)
	if !ok {
		return 0, math.Pi
	}

	best, bestAlong := uint32(0), -2.0
	for id := uint32(1); id <= g.stats.Tiles; id++ {
		i := int(id-1) * 3
		along := float64(g.positions[i])*unit.X + float64(g.positions[i+1])*unit.Y + float64(g.positions[i+2])*unit.Z
		if along > bestAlong {
			best, bestAlong = id, along
		}
	}

	return best, math.Acos(min(bestAlong, 1))
}

func angle(a, b Vec3) float64 {
	return math.Acos(min(max(a.X*b.X+a.Y*b.Y+a.Z*b.Z, -1), 1))
}

func normalize(v Vec3) (Vec3, bool) {
	length := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	if length == 0 || math.IsNaN(length) || math.IsInf(length, 0) {
		return Vec3{}, false
	}

	return Vec3{X: v.X / length, Y: v.Y / length, Z: v.Z / length}, true
}

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
