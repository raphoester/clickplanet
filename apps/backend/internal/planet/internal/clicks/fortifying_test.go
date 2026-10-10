package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func TestAFlagHoldingAWholeLandmassMayFortifyIt(t *testing.T) {
	require.NoError(t, clicks.FortifyError(3, "fr", 12, 12, ""))
	require.NoError(t, clicks.FortifyError(3, "fr", 12, 12, "de"))
}

func TestAFlagMissingATileMayNotFortify(t *testing.T) {
	require.ErrorIs(t, clicks.FortifyError(3, "fr", 11, 12, ""), clicks.ErrNotWhole)
}

func TestNoLandmassAndNoFlagAreNeverWhole(t *testing.T) {
	require.ErrorIs(t, clicks.FortifyError(clicks.NoLandmass, "fr", 4, 4, ""), clicks.ErrNotWhole)
	require.ErrorIs(t, clicks.FortifyError(3, "", 4, 4, ""), clicks.ErrNotWhole)
	require.ErrorIs(t, clicks.FortifyError(3, "fr", 0, 0, ""), clicks.ErrNotWhole)
}

func TestTheFlagThatFortifiedLastMayNotFortifyAgain(t *testing.T) {
	require.ErrorIs(t, clicks.FortifyError(3, "fr", 12, 12, "fr"), clicks.ErrFortifiedAlready)
}

func TestAWholeLandmassNobodyFortifiedSettlesOnItsHolder(t *testing.T) {
	settler, ok := clicks.SettlerOf("fr", 12, 12, "")

	assert.True(t, ok)
	assert.Equal(t, "fr", settler)
}

func TestALandmassThatIsNotWholeOrWasFortifiedSettlesOnNobody(t *testing.T) {
	for name, settled := range map[string]func() (string, bool){
		"not whole": func() (string, bool) { return clicks.SettlerOf("fr", 11, 12, "") },
		"fortified": func() (string, bool) { return clicks.SettlerOf("fr", 12, 12, "de") },
		"nobody's":  func() (string, bool) { return clicks.SettlerOf("", 12, 12, "") },
		"no tiles":  func() (string, bool) { return clicks.SettlerOf("fr", 0, 0, "") },
	} {
		_, ok := settled()
		assert.False(t, ok, name)
	}
}

func TestEveryTileIsListedUnderItsOwnLandmass(t *testing.T) {
	borders := clicks.BordersOf(10, []uint32{2, 5, 9}, []uint32{3})

	assert.Equal(t, 3, borders.Landmasses())
	assert.Equal(t, []uint32{2, 5, 9}, borders.TilesOf(1))
	assert.Equal(t, []uint32{3}, borders.TilesOf(2))
	assert.Equal(t, []uint32{1, 4, 6, 7, 8, 10}, borders.TilesOf(clicks.NoLandmass))
	assert.Equal(t, clicks.LandmassID(1), borders.LandmassOf(5))
	assert.Equal(t, clicks.NoLandmass, borders.LandmassOf(11))
	assert.Nil(t, borders.TilesOf(3))
}
