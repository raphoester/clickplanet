package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var miner = cpsession.AccountID{0x01, 0x92, 0x6c, 0x6e, 0x7a, 0x4b, 0x7c, 0x3d, 0x8e, 0x9f, 0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x60}

const twoRings = 2

func minefield(tiles *inmemory_tile_storage.Storage) clicks.Minefield {
	return clicks.NewMinefield(tiles, line{}, tenAShield)
}

func laid(t *testing.T, tiles *inmemory_tile_storage.Storage, flag string, tile uint32) {
	t.Helper()
	_, err := minefield(tiles).Lay(t.Context(), clicks.NewMine(miner, "203.0.113.9", flag, tile))
	require.NoError(t, err)
}

// enemyTakes is a click of the flag on the tile, as a player of it makes one.
func enemyTakes(t *testing.T, tiles *inmemory_tile_storage.Storage, tile uint32, flag string) {
	t.Helper()
	ctx := cpctx.AddAccountToContext(t.Context(), "0192f5a0-7c00-7000-8000-0000000000aa")
	impact, err := clicks.NewClaiming(tiles, tenAShield).Click(ctx, tile, flag)
	require.NoError(t, err)
	require.Equal(t, clicks.Taken, impact.Outcome)
}

func detonated(t *testing.T, tiles *inmemory_tile_storage.Storage) []clicks.Detonation {
	t.Helper()
	detonations, err := minefield(tiles).Detonate(t.Context(), twoRings)
	require.NoError(t, err)
	return detonations
}

func lit(tiles *inmemory_tile_storage.Storage) bool {
	select {
	case <-tiles.Fuse():
		return true
	default:
		return false
	}
}

func TestAMineIsLaidOnATileOfItsFlagWithNoMineAndChangesNoTile(t *testing.T) {
	tiles := claimable(t, map[uint32]string{5: "fr", 6: "de"})
	changes := infections(t, tiles)

	version, err := minefield(tiles).Lay(t.Context(), clicks.NewMine(miner, "203.0.113.9", "fr", 5))
	require.NoError(t, err)

	mined, at := tiles.MinesOf("fr")
	assert.Equal(t, []uint32{5}, mined)
	assert.Equal(t, version, at, "a lay counts no version")
	assert.Empty(t, changes, "and no stream of the map hears it")
	none, _ := tiles.MinesOf("de")
	assert.Empty(t, none, "another flag has none")
}

func TestAMineOnATileNotItsFlagsOrMinedAlreadyChangesNothing(t *testing.T) {
	tiles := claimable(t, map[uint32]string{5: "fr", 6: "de"})
	laid(t, tiles, "fr", 5)

	for _, tile := range []uint32{5, 6, 7} {
		_, err := minefield(tiles).Lay(t.Context(), clicks.NewMine(miner, "203.0.113.9", "fr", tile))
		require.ErrorIs(t, err, clicks.ErrNotMineable, "tile %d", tile)
	}

	mined, _ := tiles.MinesOf("fr")
	assert.Equal(t, []uint32{5}, mined)
}

func TestATakeByAnotherFlagSetsTheMineOffAndItsBlastTakesTheTilesAroundForItsFlag(t *testing.T) {
	tiles := claimable(t, map[uint32]string{2: "de", 3: "de", 4: "it", 5: "fr", 6: "de", 7: "de", 8: "de"})
	laid(t, tiles, "fr", 5)
	changes := infections(t, tiles)

	enemyTakes(t, tiles, 5, "de")
	require.True(t, lit(tiles), "the take lights the fuse")
	assert.Equal(t, "de", next(t, changes).Update.Value, "the enemy's take lands first")

	detonations := detonated(t, tiles)

	assert.Equal(t, []string{"de", "fr", "fr", "fr", "fr", "fr", "de"}, owners(tiles, 2, 3, 4, 5, 6, 7, 8),
		"its tile and two rings around it, whoever held them")
	explosion := next(t, changes).Explosion
	require.NotNil(t, explosion, "one change for the whole explosion")
	require.Len(t, explosion.Blasts, 1)
	assert.Equal(t, clicks.MineSnapshot{Tile: 5, Flag: "fr", Account: miner, Scope: "203.0.113.9"}, explosion.Blasts[0].Mine)
	assert.Equal(t, []uint32{5, 4, 6, 3, 7}, explosion.Blasts[0].Taken, "nearest first")
	require.Len(t, detonations, 1)
	assert.Len(t, detonations[0].Impacts, 5)
	mined, _ := tiles.MinesOf("fr")
	assert.Empty(t, mined, "a mine goes off once")
	assert.Empty(t, detonated(t, tiles), "and the next explosion finds none")
}

func TestABlastLeavesAShieldedTileAndItsFlagsOwnTilesAsTheyAre(t *testing.T) {
	tiles := claimable(t, map[uint32]string{3: "de", 4: "de", 5: "fr", 6: "fr", 7: "de"})
	shield(t, tiles, 4, "de", 2)
	laid(t, tiles, "fr", 5)

	enemyTakes(t, tiles, 5, "de")
	detonations := detonated(t, tiles)

	assert.Equal(t, []string{"fr", "de", "fr", "fr", "fr"}, owners(tiles, 3, 4, 5, 6, 7))
	assert.Equal(t, 2, tiles.Shields(4), "a blast strikes no shield")
	require.Len(t, detonations, 1)
	taken := make([]uint32, 0, len(detonations[0].Impacts))
	for _, impact := range detonations[0].Impacts {
		taken = append(taken, impact.Tile)
	}
	assert.Equal(t, []uint32{5, 3, 7}, taken)
}

func TestEveryMineABlastReachesGoesOffTooEachForItsOwnFlag(t *testing.T) {
	tiles := claimable(t, map[uint32]string{
		3: "de", 4: "de", 5: "fr", 6: "de", 7: "fr", 8: "de", 9: "de", 10: "it", 11: "it",
	})
	laid(t, tiles, "fr", 5)
	laid(t, tiles, "fr", 7)
	laid(t, tiles, "de", 9)
	changes := infections(t, tiles)

	enemyTakes(t, tiles, 5, "de")
	next(t, changes)
	detonated(t, tiles)

	explosion := next(t, changes).Explosion
	require.NotNil(t, explosion)
	require.Len(t, explosion.Blasts, 3, "the mines laid together go off together, and a German one in reach too")
	assert.Equal(t, []uint32{5, 7, 9}, []uint32{
		explosion.Blasts[0].Mine.Tile, explosion.Blasts[1].Mine.Tile, explosion.Blasts[2].Mine.Tile,
	})
	assert.Equal(t, []string{"fr", "fr", "de"}, []string{
		explosion.Blasts[0].Mine.Flag, explosion.Blasts[1].Mine.Flag, explosion.Blasts[2].Mine.Flag,
	})
	assert.Equal(t, []string{"fr", "fr", "fr", "fr", "de", "de", "de", "de", "de"},
		owners(tiles, 3, 4, 5, 6, 7, 8, 9, 10, 11))
	for _, flag := range []string{"fr", "de"} {
		mined, _ := tiles.MinesOf(flag)
		assert.Empty(t, mined, flag)
	}
}

func TestEveryTakeSetsAMineOffASpreadsAnEnclosuresAndAVirussToo(t *testing.T) {
	takes := map[string]func(t *testing.T, tiles *inmemory_tile_storage.Storage){
		"a spread's": func(t *testing.T, tiles *inmemory_tile_storage.Storage) {
			t.Helper()
			_, err := clicks.NewClaiming(tiles, tenAShield).Claim(t.Context(), 5, "de")
			require.NoError(t, err)
		},
		"an enclosure's": func(t *testing.T, tiles *inmemory_tile_storage.Storage) {
			t.Helper()
			_, err := clicks.NewClaiming(tiles, tenAShield).Enclose(t.Context(), clicks.Enclosure{Country: "de"}, []uint32{5})
			require.NoError(t, err)
		},
		"a virus's": func(t *testing.T, tiles *inmemory_tile_storage.Storage) {
			t.Helper()
			_, _, err := clicks.NewContagion(tiles, line{}, first, tenAShield).
				Start(t.Context(), clicks.NewVirus("v-1", carrier, "203.0.113.7", "de", 5), 32, noon)
			require.NoError(t, err)
		},
	}
	for name, take := range takes {
		t.Run(name, func(t *testing.T) {
			tiles := claimable(t, map[uint32]string{4: "de", 5: "fr", 6: "de"})
			laid(t, tiles, "fr", 5)

			take(t, tiles)

			require.True(t, lit(tiles))
			assert.Len(t, detonated(t, tiles), 1)
			assert.Equal(t, []string{"fr", "fr", "fr"}, owners(tiles, 4, 5, 6))
		})
	}
}

func TestABombTakesTheMinesItHitsOffWithNoBlastAShieldedTilesToo(t *testing.T) {
	tiles := claimable(t, map[uint32]string{5: "fr", 6: "fr", 7: "de"})
	shield(t, tiles, 6, "fr", 1)
	laid(t, tiles, "fr", 5)
	laid(t, tiles, "fr", 6)

	_, err := tiles.Clear(t.Context(), clicks.Blast{Cleared: []uint32{5, 6}})
	require.NoError(t, err)

	mined, _ := tiles.MinesOf("fr")
	assert.Empty(t, mined)
	assert.False(t, lit(tiles))
	assert.Empty(t, detonated(t, tiles))
	assert.Equal(t, []string{"", "fr", "de"}, owners(tiles, 5, 6, 7))
}

func TestAnyOtherChangeOfOwnerTakesTheMineOffWithNoBlast(t *testing.T) {
	tiles := claimable(t, map[uint32]string{5: "fr", 6: "de"})
	laid(t, tiles, "fr", 5)

	_, err := tiles.Restore(t.Context(), []clicks.Restoration{{Tile: 5, From: "fr", To: "de"}})
	require.NoError(t, err)

	mined, _ := tiles.MinesOf("fr")
	assert.Empty(t, mined, "an operator's revert")
	assert.False(t, lit(tiles))
	assert.Empty(t, detonated(t, tiles))
}

func TestABlastsTakesAreHeldForTheMinesPlayerAndFortifyALandmassItMadeWhole(t *testing.T) {
	tiles := claimable(t, map[uint32]string{21: "fr", 22: "de"})
	laid(t, tiles, "fr", 21)

	enemyTakes(t, tiles, 21, "de")
	detonations, err := minefield(tiles).Detonate(t.Context(), 1)
	require.NoError(t, err)

	assert.Equal(t, []clicks.Holding{{Account: miner, Flag: "fr", Tiles: 3}}, tiles.HoldingsOf(1))
	require.Len(t, detonations, 1)
	fortified := 0
	for _, impact := range detonations[0].Impacts {
		if impact.Fortified != nil {
			fortified++
		}
	}
	assert.Equal(t, 1, fortified, "the take that made it whole")
}

func TestAnExplosionWithNoMineTrippedSetsNothingOff(t *testing.T) {
	tiles := claimable(t, map[uint32]string{})

	detonations, err := minefield(tiles).Detonate(t.Context(), twoRings)

	require.NoError(t, err)
	assert.Empty(t, detonations)
}
