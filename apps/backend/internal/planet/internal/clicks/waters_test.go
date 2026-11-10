package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func TestWatersSayWhichBodiesEachCoastTileTouches(t *testing.T) {
	waters, err := clicks.NewWaters(10, [][]uint32{{1, 2, 5}, {5, 9}})
	require.NoError(t, err)

	assert.Equal(t, []clicks.BodyID{0}, waters.BodiesOf(1))
	assert.Equal(t, []clicks.BodyID{0, 1}, waters.BodiesOf(5), "a tile between two waters is on both")
	assert.Empty(t, waters.BodiesOf(3), "inland")
	assert.Empty(t, waters.BodiesOf(11), "off the map")
	assert.Equal(t, []uint32{5, 9}, waters.CoastOf(1))
	assert.Empty(t, waters.CoastOf(2))
	assert.Equal(t, 2, waters.Bodies())
}

func TestWatersRefuseACoastOffTheMap(t *testing.T) {
	_, err := clicks.NewWaters(10, [][]uint32{{1, 11}})
	require.ErrorContains(t, err, "off the 10-tile map")

	_, err = clicks.NewWaters(10, [][]uint32{{0}})
	require.ErrorContains(t, err, "off the 10-tile map", "tile ids start at 1")
}

func TestWatersRefuseACoastOutOfOrder(t *testing.T) {
	_, err := clicks.NewWaters(10, [][]uint32{{2, 1}})
	require.ErrorContains(t, err, "out of order")
}
