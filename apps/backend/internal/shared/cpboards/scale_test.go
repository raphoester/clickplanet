package cpboards_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
)

func TestOnEarth1EveryValueComesOutUnchanged(t *testing.T) {
	earth1, err := cpboards.ScaleOf("earth-1")
	require.NoError(t, err)

	assert.InDelta(t, 1.0, earth1.Length(), 0, "exactly: its manifest's spacing is the reference")
	assert.InDelta(t, 1.0, earth1.Density(), 0)
	for _, tiles := range []int{1, 12, 50, 150, 300, 1_000, 3_000} {
		assert.Equal(t, tiles, earth1.Tiles(tiles))
	}
	assert.InDelta(t, 20.0, earth1.Zoom(20), 0, "exactly")
	assert.InDelta(t, 0.4, earth1.Distance(0.4), 0, "exactly")
}

func TestAnEventFromBeforeTheBoardsWereNamedIsOnEarth1(t *testing.T) {
	scale, err := cpboards.ScaleOf("")

	require.NoError(t, err)
	assert.Equal(t, cpboards.Earth1(), scale)
}

func TestCountsScaleWithTheDensityAndLengthsWithTheSpacing(t *testing.T) {
	fixture, err := cpboards.ScaleOf(cpboards.Fixture)
	require.NoError(t, err)

	assert.InDelta(t, 9.697, fixture.Length(), 0.001, "the fixture's tiles are 9.7 times as far apart")
	assert.InDelta(t, 1/(9.697*9.697), fixture.Density(), 1e-5)
	assert.Equal(t, 11, fixture.Tiles(1_000), "as much ground: about a ninety-fourth of the tiles")
	assert.Equal(t, 32, fixture.Tiles(3_000))
	assert.Equal(t, 1, fixture.Tiles(50), "one tile at least")
	assert.InDelta(t, 2.062, fixture.Zoom(20), 0.001, "a tile as wide on the screen")
	assert.InDelta(t, 3.879, fixture.Distance(0.4), 0.001)
}

func TestABoardThisAppDoesNotCarryHasNoScale(t *testing.T) {
	_, err := cpboards.ScaleOf("moon-1")

	require.ErrorIs(t, err, cpboards.ErrUnknownBoard)
}

func TestTheShippedBoardsAreEarth1AndTheTestsFixtureBesideIt(t *testing.T) {
	boards, err := cpboards.Boards(cpboards.Files())
	require.NoError(t, err)

	names := make([]string, len(boards))
	for i, board := range boards {
		names[i] = board.Manifest.Name
	}
	assert.Equal(t, []string{"earth-1", cpboards.Fixture}, names)
	assert.Equal(t, "3fb28bdf", boards[0].Hash())
	assert.Equal(t, "cb8330a8", boards[1].Hash(), "another board")
}
