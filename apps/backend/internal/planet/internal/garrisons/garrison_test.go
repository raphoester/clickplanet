package garrisons_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons"
)

func TestAGarrisonStandsOnlyForItsOwnFlag(t *testing.T) {
	garrison := garrisons.Garrison{Tile: 7, Country: "fr", Defenders: 4}

	assert.Equal(t, 4, garrison.Standing("fr"))
	assert.Zero(t, garrison.Standing("de"))
	assert.Zero(t, garrisons.Garrison{Tile: 7, Defenders: 4}.Standing(""), "an empty tile has no garrison")
}

func TestReinforcingAddsOneUpToTheMost(t *testing.T) {
	garrison := garrisons.Garrison{Tile: 7}.Reinforced("fr", 2)
	require.Equal(t, garrisons.Garrison{Tile: 7, Country: "fr", Defenders: 1}, garrison)

	garrison = garrison.Reinforced("fr", 2).Reinforced("fr", 2)
	assert.Equal(t, 2, garrison.Defenders)
	assert.True(t, garrison.Full("fr", 2))
}

func TestANewFlagStartsAGarrisonOfItsOwn(t *testing.T) {
	garrison := garrisons.Garrison{Tile: 7, Country: "de", Defenders: 5}.Reinforced("fr", 10)

	assert.Equal(t, garrisons.Garrison{Tile: 7, Country: "fr", Defenders: 1}, garrison,
		"defenders left behind by a flag that lost the tile never fight for the one that took it")
}

func TestAStrikeTakesOneDefender(t *testing.T) {
	struck, hit := garrisons.Garrison{Tile: 7, Country: "fr", Defenders: 2}.Struck("fr")

	require.True(t, hit)
	assert.Equal(t, garrisons.Garrison{Tile: 7, Country: "fr", Defenders: 1}, struck)
}

func TestAStrikeOverAGarrisonStandingForNobodyEmptiesIt(t *testing.T) {
	struck, hit := garrisons.Garrison{Tile: 7, Country: "fr", Defenders: 2}.Struck("de")

	assert.False(t, hit)
	assert.True(t, struck.Empty())
	assert.Equal(t, uint32(7), struck.Tile)
}
