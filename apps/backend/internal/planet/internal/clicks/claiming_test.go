package clicks_test

import (
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func tenAShield() int { return 10 }

func shield(t *testing.T, tiles *inmemory_tile_storage.Storage, tile uint32, flag string, shields int) {
	t.Helper()
	for range shields {
		_, err := tiles.Shield(t.Context(), tile, flag, 10, clicks.PlacementID{})
		require.NoError(t, err)
	}
}

func claimable(t *testing.T, owners map[uint32]string) *inmemory_tile_storage.Storage {
	t.Helper()
	tiles := inmemory_tile_storage.New(clicks.BordersOf(100, []uint32{20, 21, 22}), clicks.NoCapitals(), inmemory_tile_storage.Config{},
		inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}), cpatomic.FakeTransactioner{}, &inmemory_tile_storage.RecordedPlacements{}, cptime.SystemClock{}, slog.New(slog.DiscardHandler))
	for tile, owner := range owners {
		require.NoError(t, tiles.Set(t.Context(), tile, owner))
	}
	return tiles
}

func TestAClaimTakesAForeignTileAndSaysWhoHeldIt(t *testing.T) {
	tiles := claimable(t, map[uint32]string{7: "de"})

	impact, err := clicks.NewClaiming(tiles, tenAShield).Claim(t.Context(), 7, "fr")
	require.NoError(t, err)

	assert.Equal(t, clicks.Impact{Tile: 7, Owner: "de", Outcome: clicks.Taken}, impact)
	owner, _ := tiles.Owner(7)
	assert.Equal(t, "fr", owner)
}

func TestAClaimOnAShieldStrikesItAndSaysHowManyAreLeft(t *testing.T) {
	tiles := claimable(t, map[uint32]string{7: "de"})
	ctx := t.Context()
	shield(t, tiles, 7, "de", 2)

	impact, err := clicks.NewClaiming(tiles, tenAShield).Click(ctx, 7, "fr")
	require.NoError(t, err)

	assert.Equal(t, clicks.Impact{Tile: 7, Owner: "de", Outcome: clicks.Shielded, Shields: 1}, impact)
	owner, _ := tiles.Owner(7)
	assert.Equal(t, "de", owner)
}

func TestAClaimOfATileTheFlagHoldsChangesNothing(t *testing.T) {
	tiles := claimable(t, map[uint32]string{7: "fr"})

	impact, err := clicks.NewClaiming(tiles, tenAShield).Claim(t.Context(), 7, "fr")
	require.NoError(t, err)

	assert.Equal(t, clicks.Impact{Tile: 7, Owner: "fr", Outcome: clicks.Unchanged}, impact)
}

func TestAClaimOffTheMapFails(t *testing.T) {
	_, err := clicks.NewClaiming(claimable(t, map[uint32]string{}), tenAShield).Claim(t.Context(), 101, "fr")

	require.Error(t, err)
}

func TestAClaimThatMakesALandmassWholeFortifiesIt(t *testing.T) {
	tiles := claimable(t, map[uint32]string{20: "fr", 21: "fr", 22: "de"})

	impact, err := clicks.NewClaiming(tiles, tenAShield).Claim(t.Context(), 22, "fr")
	require.NoError(t, err)

	require.NotNil(t, impact.Fortified)
	assert.Equal(t, clicks.LandmassID(1), impact.Fortified.Landmass)
	assert.Equal(t, "fr", impact.Fortified.Country)
	assert.Equal(t, uint32(22), impact.Fortified.Tile, "the take that made it whole")
	assert.Equal(t, 1, impact.Shields)
	assert.Equal(t, []int{1, 1, 1}, []int{tiles.Shields(20), tiles.Shields(21), tiles.Shields(22)})
}

func TestAClaimOfATileTheFlagHoldsDoesNotFortify(t *testing.T) {
	tiles := claimable(t, map[uint32]string{20: "fr", 21: "fr", 22: "fr"})

	_, err := clicks.NewClaiming(tiles, tenAShield).Claim(t.Context(), 22, "fr")
	require.NoError(t, err)

	assert.Zero(t, tiles.Shields(20))
}

func atOnce(n int, do func(i int)) {
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			<-start
			do(i)
		})
	}
	close(start)
	wg.Wait()
}

func TestClaimsOfOneTileForOneFlagAtOnceTakeItOnce(t *testing.T) {
	for range 300 {
		tiles := claimable(t, map[uint32]string{7: "de"})
		claiming := clicks.NewClaiming(tiles, tenAShield)

		outcomes := make([]clicks.Outcome, 8)
		atOnce(len(outcomes), func(i int) {
			impact, err := claiming.Click(t.Context(), 7, "fr")
			assert.NoError(t, err)
			outcomes[i] = impact.Outcome
		})

		require.Equal(t, map[clicks.Outcome]int{clicks.Taken: 1, clicks.Unchanged: 7}, tally(outcomes))
	}
}

func tally(outcomes []clicks.Outcome) map[clicks.Outcome]int {
	counts := map[clicks.Outcome]int{}
	for _, outcome := range outcomes {
		counts[outcome]++
	}
	return counts
}

func TestAShieldPlacedDuringAForeignClaimIsStruckOrRefusedButNeverLost(t *testing.T) {
	for range 1_000 {
		tiles := claimable(t, map[uint32]string{7: "de"})
		claiming := clicks.NewClaiming(tiles, tenAShield)

		var impact clicks.Impact
		placements := make([]error, 4)
		atOnce(1+len(placements), func(i int) {
			if i == len(placements) {
				var err error
				impact, err = claiming.Click(t.Context(), 7, "fr")
				assert.NoError(t, err)
				return
			}
			_, placements[i] = tiles.Shield(t.Context(), 7, "de", 10, clicks.PlacementID{})
		})

		placed := 0
		for _, err := range placements {
			if err == nil {
				placed++
				continue
			}
			require.ErrorIs(t, err, clicks.ErrNotYourTile)
		}
		owner, _ := tiles.Owner(7)
		if placed == 0 {
			require.Equal(t, clicks.Impact{Tile: 7, Owner: "de", Outcome: clicks.Taken}, impact)
			require.Equal(t, "fr", owner)
			require.Zero(t, tiles.Shields(7))
			continue
		}
		require.Equal(t, clicks.Shielded, impact.Outcome, "a shield placed before the click took it")
		require.Equal(t, "de", owner)
		require.Equal(t, placed-1, tiles.Shields(7), "every other shield placed still stands")
	}
}

func TestAnEnclosureClaimsItsTilesAsOneChangeNamingWhoClosedIt(t *testing.T) {
	tiles := claimable(t, map[uint32]string{7: "de", 8: "de"})
	shield(t, tiles, 8, "de", 1)
	listener, _, err := tiles.Subscribe(t.Context())
	require.NoError(t, err)
	ctx := cpctx.AddAccountToContext(t.Context(), "a-player")

	impacts, err := clicks.NewClaiming(tiles, tenAShield).Enclose(ctx, clicks.Enclosure{Country: "fr", ClosingTile: 6}, []uint32{7, 8, 20, 21, 22})
	require.NoError(t, err)

	require.Len(t, impacts, 5)
	assert.Equal(t, clicks.Taken, impacts[0].Outcome)
	assert.Equal(t, clicks.Shielded, impacts[1].Outcome, "a bonus is never a way around a shield")
	assert.NotNil(t, impacts[4].Fortified, "the take that made the island whole fortified it")

	enclosure := (<-listener).Enclosure
	require.NotNil(t, enclosure)
	assert.Equal(t, []uint32{7, 8, 20, 21, 22}, enclosure.Filled)
	assert.Equal(t, []clicks.TileShields{{Tile: 8, Shields: 0}}, enclosure.Struck)
	assert.Equal(t, "a-player", enclosure.By.Account)
	assert.NotNil(t, (<-listener).Fortification)
	assert.Empty(t, listener, "nothing else was published")
}
