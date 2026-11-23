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

var trapper = cpsession.AccountID{0x01, 0x92, 0x6c, 0x6e, 0x7a, 0x4b, 0x7c, 0x3d, 0x8e, 0x9f, 0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x70}

const taker = "0192f5a0-7c00-7000-8000-0000000000bb"

func trapped(t *testing.T, tiles *inmemory_tile_storage.Storage, flag string, tile uint32) {
	t.Helper()
	_, err := clicks.NewTrapline(tiles).Lay(t.Context(), clicks.NewTrap(trapper, flag, tile))
	require.NoError(t, err)
}

func TestATrapIsLaidOnATileOfItsFlagAndChangesNoTile(t *testing.T) {
	tiles := claimable(t, map[uint32]string{5: "fr", 6: "de"})
	changes := infections(t, tiles)

	version, err := clicks.NewTrapline(tiles).Lay(t.Context(), clicks.NewTrap(trapper, "fr", 5))
	require.NoError(t, err)

	traps, at := tiles.TrapsOf("fr")
	assert.Equal(t, []uint32{5}, traps)
	assert.Equal(t, version, at, "a lay counts no version")
	assert.Empty(t, changes, "and no stream of the map hears it")
	none, _ := tiles.TrapsOf("de")
	assert.Empty(t, none, "another flag has none")
}

func TestATileHoldsOneMineOrOneTrap(t *testing.T) {
	tiles := claimable(t, map[uint32]string{5: "fr", 6: "de", 7: "fr"})
	trapped(t, tiles, "fr", 5)
	laid(t, tiles, "fr", 7)

	for _, tile := range []uint32{5, 6, 7, 8} {
		_, err := clicks.NewTrapline(tiles).Lay(t.Context(), clicks.NewTrap(trapper, "fr", tile))
		require.ErrorIs(t, err, clicks.ErrNotTrappable, "tile %d", tile)
	}
	_, err := minefield(tiles).Lay(t.Context(), clicks.NewMine(miner, "203.0.113.9", false, "fr", 5))
	require.ErrorIs(t, err, clicks.ErrNotMineable, "no mine on a trap")

	traps, _ := tiles.TrapsOf("fr")
	assert.Equal(t, []uint32{5}, traps)
	mined, _ := tiles.MinesOf("fr")
	assert.Equal(t, []uint32{7}, mined)
}

func TestAClickByAnotherFlagSpringsTheTrapAndItsPlayerPays(t *testing.T) {
	tiles := claimable(t, map[uint32]string{5: "fr", 6: "de"})
	trapped(t, tiles, "fr", 5)
	snare := clicks.NewSnareLog()
	ctx := cpctx.AddIPToContext(cpctx.AddAccountToContext(t.Context(), taker), "198.51.100.7")

	impact, err := clicks.NewClaiming(tiles, tenAShield, snare).Click(ctx, 5, "de")
	require.NoError(t, err)

	assert.Equal(t, clicks.Taken, impact.Outcome, "the taker keeps the tile")
	assert.Equal(t, &clicks.TrapSnapshot{Tile: 5, Flag: "fr", Account: trapper}, impact.Sprung)
	assert.Equal(t, []clicks.Spring{{Payer: clicks.PayerOf(ctx), Trap: *impact.Sprung}}, snare.Springs(),
		"the click's player, from its request")
	traps, _ := tiles.TrapsOf("fr")
	assert.Empty(t, traps, "the trap is gone")
	assert.Equal(t, []string{"de"}, owners(tiles, 5))
	assert.False(t, lit(tiles), "no mine's fuse")
}

func TestAClickAShieldStopsSpringsNothing(t *testing.T) {
	tiles := claimable(t, map[uint32]string{5: "fr"})
	shield(t, tiles, 5, "fr", 1)
	trapped(t, tiles, "fr", 5)
	snare := clicks.NewSnareLog()

	impact, err := clicks.NewClaiming(tiles, tenAShield, snare).Click(t.Context(), 5, "de")
	require.NoError(t, err)

	assert.Equal(t, clicks.Shielded, impact.Outcome)
	assert.Nil(t, impact.Sprung)
	assert.Empty(t, snare.Springs())
	traps, _ := tiles.TrapsOf("fr")
	assert.Equal(t, []uint32{5}, traps)
}

func TestEveryTakeSpringsATrapASpreadsAnEnclosuresAVirussAndAMinesBlast(t *testing.T) {
	takes := map[string]struct {
		take  func(t *testing.T, tiles *inmemory_tile_storage.Storage, snare clicks.Snare)
		payer clicks.Payer
	}{
		"a spread's": {
			take: func(t *testing.T, tiles *inmemory_tile_storage.Storage, snare clicks.Snare) {
				t.Helper()
				_, err := clicks.NewClaiming(tiles, tenAShield, snare).Claim(cpctx.AddAccountToContext(t.Context(), taker), 5, "de")
				require.NoError(t, err)
			},
			payer: clicks.Payer{Account: taker},
		},
		"an enclosure's": {
			take: func(t *testing.T, tiles *inmemory_tile_storage.Storage, snare clicks.Snare) {
				t.Helper()
				_, err := clicks.NewClaiming(tiles, tenAShield, snare).
					Enclose(cpctx.AddAccountToContext(t.Context(), taker), clicks.Enclosure{Country: "de"}, []uint32{5})
				require.NoError(t, err)
			},
			payer: clicks.Payer{Account: taker},
		},
		"a virus's, paid by the virus's player as it used it": {
			take: func(t *testing.T, tiles *inmemory_tile_storage.Storage, snare clicks.Snare) {
				t.Helper()
				_, _, err := clicks.NewContagion(tiles, line{}, first, tenAShield, snare).
					Start(t.Context(), clicks.NewVirus("v-1", carrier, "203.0.113.7", true, "de", 5), 32, noon)
				require.NoError(t, err)
			},
			payer: clicks.Payer{Account: carrier.String(), Scope: "203.0.113.7", Linked: true},
		},
		"a mine's blast, paid by the mine's player as it laid it": {
			take: func(t *testing.T, tiles *inmemory_tile_storage.Storage, snare clicks.Snare) {
				t.Helper()
				_, err := clicks.NewMinefield(tiles, line{}, tenAShield, snare).
					Lay(t.Context(), clicks.NewMine(miner, "203.0.113.9", false, "de", 4))
				require.NoError(t, err)
				enemyTakes(t, tiles, 4, "it")
				_, err = clicks.NewMinefield(tiles, line{}, tenAShield, snare).Detonate(t.Context(), 1)
				require.NoError(t, err)
			},
			payer: clicks.Payer{Account: miner.String(), Scope: "203.0.113.9"},
		},
	}
	for name, each := range takes {
		t.Run(name, func(t *testing.T) {
			tiles := claimable(t, map[uint32]string{4: "de", 5: "fr", 6: "de"})
			trapped(t, tiles, "fr", 5)
			snare := clicks.NewSnareLog()

			each.take(t, tiles, snare)

			assert.Equal(t, []clicks.Spring{{Payer: each.payer, Trap: clicks.TrapSnapshot{Tile: 5, Flag: "fr", Account: trapper}}},
				snare.Springs())
			assert.Equal(t, "de", owners(tiles, 5)[0], "the taker keeps the tile")
			traps, _ := tiles.TrapsOf("fr")
			assert.Empty(t, traps)
		})
	}
}

func TestABombOrARevertTakesTheTrapOffAndSpringsNothing(t *testing.T) {
	tiles := claimable(t, map[uint32]string{5: "fr", 6: "fr", 7: "fr"})
	shield(t, tiles, 6, "fr", 1)
	trapped(t, tiles, "fr", 5)
	trapped(t, tiles, "fr", 6)
	trapped(t, tiles, "fr", 7)
	snare := clicks.NewSnareLog()

	_, err := tiles.Clear(t.Context(), clicks.Blast{Cleared: []uint32{5, 6}})
	require.NoError(t, err)
	_, err = tiles.Restore(t.Context(), []clicks.Restoration{{Tile: 7, From: "fr", To: "de"}})
	require.NoError(t, err)
	_, err = clicks.NewClaiming(tiles, tenAShield, snare).Click(t.Context(), 7, "it")
	require.NoError(t, err)

	traps, _ := tiles.TrapsOf("fr")
	assert.Empty(t, traps, "a bomb takes a shielded tile's too")
	assert.Empty(t, snare.Springs())
}

func TestSprungIsEveryTrapTheTakesSprangInOrder(t *testing.T) {
	first, second := clicks.TrapSnapshot{Tile: 3, Flag: "fr"}, clicks.TrapSnapshot{Tile: 9, Flag: "de"}

	sprung := clicks.Sprung([]clicks.Impact{{Tile: 3, Sprung: &first}, {Tile: 4}, {Tile: 9, Sprung: &second}})

	assert.Equal(t, []clicks.TrapSnapshot{first, second}, sprung)
	assert.Empty(t, clicks.Sprung([]clicks.Impact{{Tile: 4}}))
}
