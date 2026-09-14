package clicks_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// line is tiles 1..n in a row, each touching the one before and after; 0 and n+1 are not candidates.
func line(n uint32) ([]uint32, func(uint32) []uint32) {
	tiles := make([]uint32, 0, n)
	for tile := uint32(1); tile <= n; tile++ {
		tiles = append(tiles, tile)
	}

	return tiles, func(tile uint32) []uint32 { return []uint32{tile - 1, tile + 1} }
}

func seeded(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed)) //nolint:gosec // a test draw, not a secret.
}

func contiguous(tiles []uint32) bool {
	return int(slices.Max(tiles)-slices.Min(tiles))+1 == len(tiles)
}

func TestFullProximityGrowsOnePatch(t *testing.T) {
	candidates, neighbours := line(1000)

	for seed := range uint64(20) {
		picked := clicks.Pick(candidates, 50, 1, neighbours, seeded(seed))

		assert.Len(t, picked, 50)
		assert.True(t, contiguous(picked), "seed %d: a gap means a draw left the patch", seed)
	}
}

func TestNoProximityScattersTheTiles(t *testing.T) {
	candidates, neighbours := line(1000)

	picked := clicks.Pick(candidates, 50, 0, neighbours, seeded(1))

	assert.Len(t, picked, 50)
	assert.False(t, contiguous(picked))
}

func TestFullProximityJumpsWhenThePatchIsWalledIn(t *testing.T) {
	// Two islands of three: once one is taken, the only way to go on is a fresh draw.
	candidates := []uint32{1, 2, 3, 11, 12, 13}
	neighbours := func(tile uint32) []uint32 { return []uint32{tile - 1, tile + 1} }

	picked := clicks.Pick(candidates, 6, 1, neighbours, seeded(7))

	assert.ElementsMatch(t, candidates, picked)
}

func TestItNeverPicksATileTwiceNorMoreThanThereAre(t *testing.T) {
	candidates, neighbours := line(30)

	picked := clicks.Pick(candidates, 100, 0.5, neighbours, seeded(3))

	assert.ElementsMatch(t, candidates, picked)
}
