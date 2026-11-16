package clicks_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var (
	carrier = cpsession.AccountID{0x01, 0x92, 0x6c, 0x6e, 0x7a, 0x4b, 0x7c, 0x3d, 0x8e, 0x9f, 0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x5f}
	noon    = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
)

func aimedAt(tile uint32) clicks.Virus {
	return clicks.NewVirus("v-1", carrier, "203.0.113.7", "fr", tile)
}

func tileOf(id uint32, owner string, shields int) clicks.Tile {
	return clicks.TileSnapshot{ID: id, Owner: owner, Shields: shields}.Restore()
}

func TestAVirusHitsTheFlagOfTheEnemyTileItIsAimedAt(t *testing.T) {
	virus, err := aimedAt(7).Aimed(tileOf(7, "de", 0))
	require.NoError(t, err)

	assert.Equal(t, "de", virus.TakeSnapshot().Victim)
}

func TestAVirusTakesNoTileOfItsOwnFlagNoBlankTileAndNoShieldedTile(t *testing.T) {
	for name, tile := range map[string]clicks.Tile{
		"its own flag's": tileOf(7, "fr", 0),
		"a blank tile":   tileOf(7, "", 0),
		"a shielded one": tileOf(7, "de", 1),
		"another tile":   tileOf(8, "de", 0),
	} {
		_, err := aimedAt(7).Aimed(tile)
		require.ErrorIs(t, err, clicks.ErrNotInfectable, name)
	}
}

func TestEveryInfectionCountsAndATakeInfectsItsTile(t *testing.T) {
	virus := aimedAt(7).
		Infected(clicks.Impact{Tile: 7, Outcome: clicks.Taken}).
		Infected(clicks.Impact{Tile: 8, Outcome: clicks.Shielded, Shields: 2})

	kept := virus.TakeSnapshot()
	assert.Equal(t, 2, kept.Infections, "a struck shield is an infection too")
	assert.Equal(t, []uint32{7}, kept.Infected)
}

func TestAVirusEndsWhenItInfectedNothingHoldsNoTileOrMadeItsMostInfections(t *testing.T) {
	alive := aimedAt(7).Infected(clicks.Impact{Tile: 7, Outcome: clicks.Taken})

	assert.False(t, alive.Ticked(true, 32, noon).TakeSnapshot().Ended)
	assert.True(t, alive.Ticked(false, 32, noon).TakeSnapshot().Ended, "it infected nothing")
	assert.True(t, alive.Cured(7).Ticked(true, 32, noon).TakeSnapshot().Ended, "its every tile was cured")
	assert.True(t, alive.Ticked(true, 1, noon).TakeSnapshot().Ended, "its most infections made")
}

func TestAVirusIsDueAtItsNextTickUntilItEnds(t *testing.T) {
	virus := aimedAt(7).Infected(clicks.Impact{Tile: 7, Outcome: clicks.Taken}).Ticked(true, 32, noon)

	assert.False(t, virus.Due(noon.Add(-time.Second)))
	assert.True(t, virus.Due(noon))
	assert.False(t, virus.Ticked(false, 32, noon).Due(noon.Add(time.Hour)), "an ended virus is never due")
}

func TestACureTakesTheTileOutOfTheVirusAlone(t *testing.T) {
	virus := aimedAt(7).
		Infected(clicks.Impact{Tile: 7, Outcome: clicks.Taken}).
		Infected(clicks.Impact{Tile: 8, Outcome: clicks.Taken})

	cured := virus.Cured(7)

	assert.Equal(t, []uint32{8}, cured.TakeSnapshot().Infected)
	assert.Equal(t, []uint32{7, 8}, virus.TakeSnapshot().Infected, "a value: the virus before is as it was")
	assert.Equal(t, 2, cured.TakeSnapshot().Infections, "a cure takes back no infection")
}
