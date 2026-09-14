package paint_random_tiles_usecase_test

import (
	"context"
	"errors"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/paint_random_tiles_usecase"
)

// Tiles 1..8 in a row. 1..6 sit on "fr" ground, 7..8 on "bg".
var ground = []string{"", "fr", "fr", "fr", "fr", "fr", "fr", "bg", "bg"}

type borders struct{}

func (borders) CountryOf(tile uint32) string { return ground[tile] }

func (borders) Tiles() uint32 { return uint32(len(ground) - 1) } //nolint:gosec // test data is tiny.

type row struct{}

func (row) Neighbours(id uint32) []uint32 { return []uint32{id - 1, id + 1} }

type stubMap struct {
	tiles   []string
	batches [][]clicks.Restoration
	// retake is set on a tile by someone else just before the first batch is painted.
	retake map[uint32]string
	err    error
}

func (m *stubMap) Owner(tile uint32) (string, bool) { return m.tiles[tile], m.tiles[tile] != "" }

func (m *stubMap) Restore(_ context.Context, restorations []clicks.Restoration) (int, error) {
	m.batches = append(m.batches, restorations)
	if m.err != nil {
		return 0, m.err
	}
	for tile, owner := range m.retake {
		m.tiles[tile] = owner
	}
	m.retake = nil

	painted := 0
	for _, r := range restorations {
		if m.tiles[r.Tile] == r.From {
			m.tiles[r.Tile] = r.To
			painted++
		}
	}
	return painted, nil
}

type countries struct{}

func (countries) CheckCountry(country string) bool {
	return country == "dz" || country == "fr" || country == "bg"
}

func newMap() *stubMap {
	return &stubMap{tiles: []string{"", "", "dz", "bg", "", "dz", "", "", "dz"}}
}

func newUseCase(tiles *stubMap, pace clicks.Pacing) *paint_random_tiles_usecase.UseCase {
	random := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // a test draw, not a secret.
	return paint_random_tiles_usecase.New(borders{}, row{}, tiles, countries{}, random, pace)
}

func TestItPaintsOnlyTheAreaAndSkipsTilesAlreadyWearingTheFlag(t *testing.T) {
	tiles := newMap()

	out, err := newUseCase(tiles, clicks.Pacing{Batch: 2}).Execute(t.Context(),
		paint_random_tiles_usecase.In{Flag: "dz", Area: "fr", Count: 10, Proximity: 0.5})
	require.NoError(t, err)

	assert.Equal(t, paint_random_tiles_usecase.Out{Eligible: 4, Picked: 4, Painted: 4}, out)
	assert.Equal(t, []string{"", "dz", "dz", "dz", "dz", "dz", "dz", "", "dz"}, tiles.tiles, "bg ground is left alone")
	assert.Len(t, tiles.batches, 2, "four tiles, two per batch")
}

func TestItPicksNoMoreThanAsked(t *testing.T) {
	tiles := newMap()

	out, err := newUseCase(tiles, clicks.Pacing{Batch: 10}).Execute(t.Context(),
		paint_random_tiles_usecase.In{Flag: "dz", Area: "fr", Count: 2, Proximity: 1})
	require.NoError(t, err)

	assert.Equal(t, paint_random_tiles_usecase.Out{Eligible: 4, Picked: 2, Painted: 2}, out)
}

func TestATileRetakenAfterThePickIsNotPainted(t *testing.T) {
	tiles := newMap()
	tiles.retake = map[uint32]string{1: "bg"}

	out, err := newUseCase(tiles, clicks.Pacing{Batch: 10}).Execute(t.Context(),
		paint_random_tiles_usecase.In{Flag: "dz", Area: "fr", Count: 4})
	require.NoError(t, err)

	assert.Equal(t, 4, out.Picked)
	assert.Equal(t, 3, out.Painted)
	assert.Equal(t, "bg", tiles.tiles[1])
}

func TestADryRunPicksAndPaintsNothing(t *testing.T) {
	tiles := newMap()

	out, err := newUseCase(tiles, clicks.Pacing{Batch: 2}).Execute(t.Context(),
		paint_random_tiles_usecase.In{Flag: "dz", Area: "fr", Count: 3, DryRun: true})
	require.NoError(t, err)

	assert.Equal(t, paint_random_tiles_usecase.Out{Eligible: 4, Picked: 3}, out)
	assert.Empty(t, tiles.batches)
}

func TestItRefusesABadRequest(t *testing.T) {
	for name, tc := range map[string]struct {
		in   paint_random_tiles_usecase.In
		want error
	}{
		"unknown flag":       {paint_random_tiles_usecase.In{Flag: "xx", Area: "fr", Count: 1}, clicks.ErrUnknownCountry},
		"no area":            {paint_random_tiles_usecase.In{Flag: "dz", Count: 1}, clicks.ErrUnknownCountry},
		"no count":           {paint_random_tiles_usecase.In{Flag: "dz", Area: "fr"}, paint_random_tiles_usecase.ErrInvalidCount},
		"proximity above 1":  {paint_random_tiles_usecase.In{Flag: "dz", Area: "fr", Count: 1, Proximity: 1.5}, paint_random_tiles_usecase.ErrInvalidProximity},
		"negative proximity": {paint_random_tiles_usecase.In{Flag: "dz", Area: "fr", Count: 1, Proximity: -0.1}, paint_random_tiles_usecase.ErrInvalidProximity},
	} {
		_, err := newUseCase(newMap(), clicks.Pacing{Batch: 2}).Execute(t.Context(), tc.in)
		require.ErrorIs(t, err, tc.want, name)
	}
}

func TestItStopsWhenTheContextEndsAndSaysHowFarItGot(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	out, err := newUseCase(newMap(), clicks.Pacing{Batch: 2, Pause: time.Hour}).Execute(ctx,
		paint_random_tiles_usecase.In{Flag: "dz", Area: "fr", Count: 4})
	require.ErrorIs(t, err, context.Canceled)

	assert.Equal(t, 2, out.Painted, "the first batch went before the pause noticed")
}

func TestAStorageErrorIsReturned(t *testing.T) {
	tiles := newMap()
	tiles.err = errors.New("code table full")

	_, err := newUseCase(tiles, clicks.Pacing{Batch: 2}).Execute(t.Context(),
		paint_random_tiles_usecase.In{Flag: "dz", Area: "fr", Count: 1})
	require.ErrorIs(t, err, tiles.err)
}
