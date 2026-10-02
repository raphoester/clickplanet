//go:build testing

package clicks

import (
	"slices"
	"sync"
)

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
		if len(frontier) == 0 {
			break
		}
		found = append(found, frontier...)
	}

	slices.Sort(found)

	return found
}

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

type walker struct {
	seen       []uint32
	generation uint32
}

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
	if w.generation == ^uint32(0) {
		clear(w.seen)
		w.generation = 0
	}
	w.generation++
}

func (w *walker) visit(id uint32) bool {
	if w.seen[id] == w.generation {
		return false
	}
	w.seen[id] = w.generation
	return true
}
