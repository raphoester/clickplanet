package clicks

import (
	"math/rand/v2"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// Random is the draw a pick makes; SystemRandom and a seeded *rand.Rand both satisfy it.
type Random interface {
	IntN(n int) int
	Float64() float64
}

// SystemRandom is math/rand/v2's global source, which is safe for concurrent calls.
type SystemRandom struct{}

func (SystemRandom) IntN(n int) int { return rand.IntN(n) } //nolint:gosec // an operator's pick, not a secret.

func (SystemRandom) Float64() float64 { return rand.Float64() } //nolint:gosec // an operator's pick, not a secret.

// Pick draws count tiles: with probability proximity an eligible tile touching one picked, else a seed.
func Pick(
	seeds []uint32,
	count int,
	proximity float64,
	neighbours func(uint32) []uint32,
	eligible func(uint32) bool,
	random Random,
) []uint32 {
	pool := newDrawSet(seeds)
	frontier := newDrawSet(nil)
	taken := cpcolls.NewSetWithCapacity[uint32](min(count, len(seeds)))
	picked := make([]uint32, 0, min(count, len(seeds)))

	for len(picked) < count && (pool.len() > 0 || frontier.len() > 0) {
		var tile uint32
		if frontier.len() > 0 && (pool.len() == 0 || random.Float64() < proximity) {
			tile = frontier.at(random.IntN(frontier.len()))
		} else {
			tile = pool.at(random.IntN(pool.len()))
		}

		pool.remove(tile)
		frontier.remove(tile)
		taken.Add(tile)
		picked = append(picked, tile)

		for _, next := range neighbours(tile) {
			if !taken.Contains(next) && !frontier.has(next) && eligible(next) {
				frontier.add(next)
			}
		}
	}

	return picked
}

// drawSet is a set with a uniform draw: a slice to index into and a map to remove from it in O(1).
type drawSet struct {
	tiles []uint32
	index map[uint32]int
}

func newDrawSet(tiles []uint32) *drawSet {
	set := &drawSet{tiles: make([]uint32, 0, len(tiles)), index: make(map[uint32]int, len(tiles))}
	for _, tile := range tiles {
		set.add(tile)
	}

	return set
}

func (s *drawSet) len() int { return len(s.tiles) }

func (s *drawSet) at(i int) uint32 { return s.tiles[i] }

func (s *drawSet) has(tile uint32) bool {
	_, ok := s.index[tile]
	return ok
}

func (s *drawSet) add(tile uint32) {
	if s.has(tile) {
		return
	}
	s.index[tile] = len(s.tiles)
	s.tiles = append(s.tiles, tile)
}

func (s *drawSet) remove(tile uint32) {
	i, ok := s.index[tile]
	if !ok {
		return
	}

	last := s.tiles[len(s.tiles)-1]
	s.tiles[i] = last
	s.index[last] = i
	s.tiles = s.tiles[:len(s.tiles)-1]
	delete(s.index, tile)
}
