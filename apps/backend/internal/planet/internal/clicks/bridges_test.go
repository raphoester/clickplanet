package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func TestBridgesSayTheFarEndOfEachBridgeOnATile(t *testing.T) {
	bridges, err := clicks.NewBridges(10, [][2]uint32{{1, 6}, {6, 9}})
	require.NoError(t, err)

	assert.Equal(t, []uint32{6}, bridges.Across(1))
	assert.Equal(t, []uint32{1, 9}, bridges.Across(6), "a tile two bridges land on")
	assert.Empty(t, bridges.Across(3), "no bridge")
	assert.Empty(t, bridges.Across(11), "off the map")
	assert.Equal(t, 2, bridges.Count())
}

func TestBridgesRefuseAnEndOffTheMap(t *testing.T) {
	_, err := clicks.NewBridges(10, [][2]uint32{{1, 11}})
	require.ErrorContains(t, err, "off the 10-tile map")

	_, err = clicks.NewBridges(10, [][2]uint32{{0, 1}})
	require.ErrorContains(t, err, "off the 10-tile map", "tile ids start at 1")
}

func TestBridgesRefuseABridgeToItsOwnTile(t *testing.T) {
	_, err := clicks.NewBridges(10, [][2]uint32{{4, 4}})
	require.ErrorContains(t, err, "both ends on tile 4")
}
