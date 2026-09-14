package clicks_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// row is tiles in a line, each touching the one before and after.
func row(tile uint32) []uint32 { return []uint32{tile - 1, tile + 1} }

func span(from, to uint32) []uint32 {
	tiles := make([]uint32, 0, to-from+1)
	for tile := from; tile <= to; tile++ {
		tiles = append(tiles, tile)
	}

	return tiles
}

func in(tiles []uint32) func(uint32) bool {
	return func(tile uint32) bool { return slices.Contains(tiles, tile) }
}

func between(from, to uint32) func(uint32) bool {
	return func(tile uint32) bool { return tile >= from && tile <= to }
}

func seeded(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed)) //nolint:gosec // a test draw, not a secret.
}

func contiguous(tiles []uint32) bool {
	return int(slices.Max(tiles)-slices.Min(tiles))+1 == len(tiles)
}

func TestFullProximityGrowsOnePatch(t *testing.T) {
	seeds := span(1, 1000)

	for seed := range uint64(20) {
		picked := clicks.Pick(seeds, 50, 1, row, in(seeds), seeded(seed))

		assert.Len(t, picked, 50)
		assert.True(t, contiguous(picked), "seed %d: a gap means a draw left the patch", seed)
	}
}

func TestNoProximityScattersTheTiles(t *testing.T) {
	seeds := span(1, 1000)

	picked := clicks.Pick(seeds, 50, 0, row, in(seeds), seeded(1))

	assert.Len(t, picked, 50)
	assert.False(t, contiguous(picked))
}

func TestAPatchGrowsPastTheSeeds(t *testing.T) {
	seeds := span(500, 509)

	picked := clicks.Pick(seeds, 50, 1, row, between(1, 1000), seeded(4))

	assert.Len(t, picked, 50)
	assert.True(t, contiguous(picked))
	assert.Less(t, slices.Min(picked), uint32(500))
	assert.Greater(t, slices.Max(picked), uint32(509))
}

func TestNoProximityStaysOnTheSeedsWhileThereAreAny(t *testing.T) {
	seeds := span(500, 509)

	picked := clicks.Pick(seeds, 10, 0, row, between(1, 1000), seeded(4))

	assert.ElementsMatch(t, seeds, picked)
}

func TestAPatchNeverGrowsIntoATileThatIsNotEligible(t *testing.T) {
	seeds := span(500, 509)

	picked := clicks.Pick(seeds, 100, 1, row, between(495, 514), seeded(4))

	assert.ElementsMatch(t, span(495, 514), picked)
}

func TestFullProximityJumpsWhenThePatchIsWalledIn(t *testing.T) {
	seeds := []uint32{1, 2, 3, 11, 12, 13}

	picked := clicks.Pick(seeds, 6, 1, row, in(seeds), seeded(7))

	assert.ElementsMatch(t, seeds, picked)
}

func TestItNeverPicksATileTwiceNorMoreThanThereAre(t *testing.T) {
	seeds := span(1, 30)

	picked := clicks.Pick(seeds, 100, 0.5, row, in(seeds), seeded(3))

	assert.ElementsMatch(t, seeds, picked)
}
