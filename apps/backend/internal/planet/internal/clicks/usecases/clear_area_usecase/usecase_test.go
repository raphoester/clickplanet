package clear_area_usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/clear_area_usecase"
)

type stubMap struct {
	owners  map[uint32]string
	batches []int
}

func (m *stubMap) Owner(tile uint32) (string, bool) { return m.owners[tile], true }

func (m *stubMap) Restore(_ context.Context, restorations []clicks.Restoration) (int, error) {
	m.batches = append(m.batches, len(restorations))
	cleared := 0
	for _, r := range restorations {
		if m.owners[r.Tile] == r.From {
			m.owners[r.Tile] = r.To
			cleared++
		}
	}
	return cleared, nil
}

type stubAreas struct {
	area []uint32
	err  error
}

func (a stubAreas) Around(uint32, uint32) ([]uint32, error) { return a.area, a.err }

func TestEveryOwnedTileOfTheAreaGoesToNobodyInBatches(t *testing.T) {
	tiles := &stubMap{owners: map[uint32]string{1: "de", 2: "", 3: "de", 4: "fr", 5: "de"}}

	out, err := clear_area_usecase.New(tiles, stubAreas{area: []uint32{1, 2, 3, 4}}, clicks.Pacing{Batch: 2}).
		Execute(t.Context(), clear_area_usecase.In{Tile: 1, Rings: 1})
	require.NoError(t, err)

	assert.Equal(t, clear_area_usecase.Out{Tiles: 4, Owned: 3, Cleared: 3}, out)
	assert.Equal(t, map[uint32]string{1: "", 2: "", 3: "", 4: "", 5: "de"}, tiles.owners)
	assert.Equal(t, []int{2, 1}, tiles.batches)
}

func TestADryRunCountsAndClearsNothing(t *testing.T) {
	tiles := &stubMap{owners: map[uint32]string{1: "de"}}

	out, err := clear_area_usecase.New(tiles, stubAreas{area: []uint32{1, 2}}, clicks.Pacing{Batch: 2}).
		Execute(t.Context(), clear_area_usecase.In{Tile: 1, Rings: 1, DryRun: true})
	require.NoError(t, err)

	assert.Equal(t, clear_area_usecase.Out{Tiles: 2, Owned: 1}, out)
	assert.Equal(t, "de", tiles.owners[1])
}

func TestAnAreaThatIsNotOneIsRefused(t *testing.T) {
	_, err := clear_area_usecase.New(&stubMap{}, stubAreas{err: clicks.ErrInvalidArea}, clicks.Pacing{Batch: 2}).
		Execute(t.Context(), clear_area_usecase.In{Tile: 1, Rings: 99})

	require.ErrorIs(t, err, clicks.ErrInvalidArea)
}
