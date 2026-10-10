package clicks_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
)

func claimable(t *testing.T, owners map[uint32]string) *inmemory_tile_storage.Storage {
	t.Helper()
	tiles := inmemory_tile_storage.New(clicks.BordersOf(100, []uint32{20, 21, 22}), inmemory_tile_storage.Config{},
		inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}), slog.New(slog.DiscardHandler))
	for tile, owner := range owners {
		require.NoError(t, tiles.Set(t.Context(), tile, owner))
	}
	return tiles
}

func TestAClaimTakesAForeignTileAndSaysWhoHeldIt(t *testing.T) {
	tiles := claimable(t, map[uint32]string{7: "de"})

	impact, err := clicks.NewClaiming(tiles, 10).Claim(t.Context(), 7, "fr")
	require.NoError(t, err)

	assert.Equal(t, clicks.Impact{Tile: 7, Owner: "de", Outcome: clicks.Taken}, impact)
	owner, _ := tiles.Owner(7)
	assert.Equal(t, "fr", owner)
}

func TestAClaimOnAShieldStrikesItAndSaysHowManyAreLeft(t *testing.T) {
	tiles := claimable(t, map[uint32]string{7: "de"})
	ctx := t.Context()
	require.NoError(t, tiles.Shield(ctx, 7, "de", 10))
	require.NoError(t, tiles.Shield(ctx, 7, "de", 10))

	impact, err := clicks.NewClaiming(tiles, 10).Click(ctx, 7, "fr")
	require.NoError(t, err)

	assert.Equal(t, clicks.Impact{Tile: 7, Owner: "de", Outcome: clicks.Shielded, Shields: 1}, impact)
	owner, _ := tiles.Owner(7)
	assert.Equal(t, "de", owner)
}

func TestAClaimOfATileTheFlagHoldsChangesNothing(t *testing.T) {
	tiles := claimable(t, map[uint32]string{7: "fr"})

	impact, err := clicks.NewClaiming(tiles, 10).Claim(t.Context(), 7, "fr")
	require.NoError(t, err)

	assert.Equal(t, clicks.Impact{Tile: 7, Owner: "fr", Outcome: clicks.Unchanged}, impact)
}

func TestAClaimOffTheMapFails(t *testing.T) {
	_, err := clicks.NewClaiming(claimable(t, map[uint32]string{}), 10).Claim(t.Context(), 101, "fr")

	require.Error(t, err)
}

func TestAClaimThatMakesALandmassWholeFortifiesIt(t *testing.T) {
	tiles := claimable(t, map[uint32]string{20: "fr", 21: "fr", 22: "de"})

	impact, err := clicks.NewClaiming(tiles, 10).Claim(t.Context(), 22, "fr")
	require.NoError(t, err)

	require.NotNil(t, impact.Fortified)
	assert.Equal(t, clicks.LandmassID(1), impact.Fortified.Landmass)
	assert.Equal(t, "fr", impact.Fortified.Country)
	assert.Equal(t, 1, impact.Shields)
	assert.Equal(t, []int{1, 1, 1}, []int{tiles.Shields(20), tiles.Shields(21), tiles.Shields(22)})
}

func TestAClaimOfATileTheFlagHoldsDoesNotFortify(t *testing.T) {
	tiles := claimable(t, map[uint32]string{20: "fr", 21: "fr", 22: "fr"})

	_, err := clicks.NewClaiming(tiles, 10).Claim(t.Context(), 22, "fr")
	require.NoError(t, err)

	assert.Zero(t, tiles.Shields(20))
}
