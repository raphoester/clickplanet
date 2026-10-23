package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type owners map[uint32]string

func (o owners) Owner(tile uint32) (string, bool) { return o[tile], true }

func TestAnAreaIsEveryTileWithinItsRings(t *testing.T) {
	areas := clicks.NewAreas(equatorRow(t), 0.11)

	one, err := areas.Around(1, 1)
	require.NoError(t, err)
	assert.Equal(t, []uint32{1, 2}, one)

	two, err := areas.Around(1, 2)
	require.NoError(t, err)
	assert.Equal(t, []uint32{1, 2, 3}, two)
}

func TestAnAreaOfNoRingsTooManyOrNoTileIsRefused(t *testing.T) {
	areas := clicks.NewAreas(equatorRow(t), 0.11)

	for _, rings := range []uint32{0, clicks.MaxAreaRings + 1} {
		_, err := areas.Around(1, rings)
		require.ErrorIs(t, err, clicks.ErrInvalidArea, rings)
	}

	_, err := areas.Around(4, 1)
	require.ErrorIs(t, err, clicks.ErrTileOutOfRange)
}

func TestEmptyingGivesEveryOwnedTileToNobodyAgainstItsOwnerNow(t *testing.T) {
	restorations := clicks.Emptying([]uint32{1, 2, 3}, owners{1: "de", 3: "fr"})

	assert.Equal(t, []clicks.Restoration{{Tile: 1, From: "de"}, {Tile: 3, From: "fr"}}, restorations)
}
