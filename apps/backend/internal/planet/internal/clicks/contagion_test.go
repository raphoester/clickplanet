package clicks_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
)

// A line of tiles: each touches the one before it and the one after it.
type line struct{}

func (line) Neighbours(id uint32) []uint32 {
	if id <= 1 {
		return []uint32{id + 1}
	}
	return []uint32{id - 1, id + 1}
}

func first(int) int { return 0 }

func contagion(tiles *inmemory_tile_storage.Storage) clicks.Contagion {
	return clicks.NewContagion(tiles, line{}, first, tenAShield)
}

func owners(tiles *inmemory_tile_storage.Storage, ids ...uint32) []string {
	flags := make([]string, len(ids))
	for i, id := range ids {
		flags[i], _ = tiles.Owner(id)
	}
	return flags
}

func infections(t *testing.T, tiles *inmemory_tile_storage.Storage) <-chan clicks.Change {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	changes, _, err := tiles.Subscribe(ctx)
	require.NoError(t, err)
	return changes
}

func next(t *testing.T, changes <-chan clicks.Change) clicks.Change {
	t.Helper()
	select {
	case change := <-changes:
		return change
	case <-time.After(time.Second):
		require.FailNow(t, "no change came")
		return clicks.Change{}
	}
}

func started(t *testing.T, tiles *inmemory_tile_storage.Storage, tile uint32, most int) clicks.Virus {
	t.Helper()
	virus, impacts, err := contagion(tiles).Start(t.Context(), aimedAt(tile), most, noon)
	require.NoError(t, err)
	require.Len(t, impacts, 1)
	return virus
}

func TestAVirusTakesTheTileItIsAimedAtAndInfectsItAsOneChange(t *testing.T) {
	tiles := claimable(t, map[uint32]string{5: "de", 6: "de"})
	changes := infections(t, tiles)

	virus := started(t, tiles, 5, 32)

	assert.Equal(t, []string{"fr", "de"}, owners(tiles, 5, 6))
	infection := next(t, changes).Infection
	require.NotNil(t, infection)
	assert.Equal(t, []uint32{5}, infection.Taken)
	assert.Equal(t, clicks.VirusSnapshot{
		ID: "v-1", Account: carrier, Scope: "203.0.113.7", Flag: "fr", Victim: "de", Tile: 5, Infected: []uint32{5},
		Infections: 1, Next: noon,
	}, infection.Virus)
	assert.Equal(t, []clicks.Virus{virus}, tiles.Viruses())
}

func TestAVirusAimedAtNoEnemyTileOrAShieldedOneChangesNothing(t *testing.T) {
	tiles := claimable(t, map[uint32]string{5: "de", 6: "fr"})
	shield(t, tiles, 5, "de", 1)

	for _, tile := range []uint32{5, 6, 7} {
		_, _, err := contagion(tiles).Start(t.Context(), aimedAt(tile), 32, noon)
		require.ErrorIs(t, err, clicks.ErrNotInfectable, "tile %d", tile)
	}

	assert.Equal(t, []string{"de", "fr", ""}, owners(tiles, 5, 6, 7))
	assert.Equal(t, 1, tiles.Shields(5))
	assert.Empty(t, tiles.Viruses())
}

func TestEachInfectedTileInfectsOneTileOfTheFlagHitAtEachTick(t *testing.T) {
	tiles := claimable(t, map[uint32]string{3: "de", 4: "de", 5: "de", 6: "de", 7: "de"})
	started(t, tiles, 5, 32)

	_, impacts, err := contagion(tiles).Spread(t.Context(), "v-1", 32, noon)
	require.NoError(t, err)
	assert.Len(t, impacts, 1, "one tile infected, one take")
	assert.Equal(t, []string{"de", "fr", "fr", "de", "de"}, owners(tiles, 3, 4, 5, 6, 7))

	virus, impacts, err := contagion(tiles).Spread(t.Context(), "v-1", 32, noon)
	require.NoError(t, err)
	assert.Len(t, impacts, 2, "two tiles infected, two takes: it doubles")
	assert.Equal(t, []string{"fr", "fr", "fr", "fr", "de"}, owners(tiles, 3, 4, 5, 6, 7))
	assert.Equal(t, []uint32{5, 4, 6, 3}, virus.TakeSnapshot().Infected)
	assert.Equal(t, 4, virus.TakeSnapshot().Infections)
}

func TestAVirusSpreadsOnlyIntoTheLandOfTheFlagItHit(t *testing.T) {
	tiles := claimable(t, map[uint32]string{4: "it", 5: "de", 6: "de"})
	started(t, tiles, 5, 32)

	_, _, err := contagion(tiles).Spread(t.Context(), "v-1", 32, noon)
	require.NoError(t, err)

	assert.Equal(t, []string{"it", "fr", "fr"}, owners(tiles, 4, 5, 6))
}

func TestAShieldAbsorbsAnInfectionWhichCountsTowardTheMost(t *testing.T) {
	tiles := claimable(t, map[uint32]string{4: "de", 5: "de"})
	shield(t, tiles, 4, "de", 2)
	started(t, tiles, 5, 32)
	changes := infections(t, tiles)

	virus, _, err := contagion(tiles).Spread(t.Context(), "v-1", 32, noon)
	require.NoError(t, err)

	assert.Equal(t, []string{"de", "fr"}, owners(tiles, 4, 5))
	assert.Equal(t, 1, tiles.Shields(4))
	assert.Equal(t, 2, virus.TakeSnapshot().Infections)
	infection := next(t, changes).Infection
	require.NotNil(t, infection)
	assert.Equal(t, []clicks.TileShields{{Tile: 4, Shields: 1}}, infection.Struck)
	assert.Empty(t, infection.Taken)
}

func TestAVirusStopsAtItsMostInfectionsAndEnds(t *testing.T) {
	tiles := claimable(t, map[uint32]string{3: "de", 4: "de", 5: "de", 6: "de", 7: "de"})
	started(t, tiles, 5, 3)
	_, _, err := contagion(tiles).Spread(t.Context(), "v-1", 3, noon)
	require.NoError(t, err)
	changes := infections(t, tiles)

	virus, impacts, err := contagion(tiles).Spread(t.Context(), "v-1", 3, noon)
	require.NoError(t, err)

	assert.Len(t, impacts, 1, "the third infection is the last")
	assert.True(t, virus.TakeSnapshot().Ended)
	assert.Empty(t, tiles.Viruses(), "an ended virus leaves the map")
	infection := next(t, changes).Infection
	require.NotNil(t, infection)
	assert.True(t, infection.Virus.Ended)
}

func TestATileTakenBackIsCuredAndSpreadsNoMore(t *testing.T) {
	tiles := claimable(t, map[uint32]string{3: "de", 4: "de", 5: "de"})
	started(t, tiles, 5, 32)
	_, _, err := contagion(tiles).Spread(t.Context(), "v-1", 32, noon)
	require.NoError(t, err)

	_, err = clicks.NewClaiming(tiles, tenAShield).Click(t.Context(), 4, "de")
	require.NoError(t, err)

	assert.Equal(t, []uint32{5}, tiles.Viruses()[0].TakeSnapshot().Infected)
	_, _, err = contagion(tiles).Spread(t.Context(), "v-1", 32, noon)
	require.NoError(t, err)
	assert.Equal(t, []string{"de", "fr", "fr"}, owners(tiles, 3, 4, 5), "only tile 5 infected again, next to it")
}

func TestAVirusWithNowhereToGoEnds(t *testing.T) {
	tiles := claimable(t, map[uint32]string{4: "fr", 5: "de", 6: "it"})
	started(t, tiles, 5, 32)
	changes := infections(t, tiles)

	virus, impacts, err := contagion(tiles).Spread(t.Context(), "v-1", 32, noon)
	require.NoError(t, err)

	assert.Empty(t, impacts)
	assert.True(t, virus.TakeSnapshot().Ended)
	assert.Empty(t, tiles.Viruses())
	infection := next(t, changes).Infection
	require.NotNil(t, infection, "its end is told, so every page stops drawing it")
	assert.True(t, infection.Virus.Ended)
}

func TestAVirusThatLeftTheMapIsNoVirus(t *testing.T) {
	_, _, err := contagion(claimable(t, map[uint32]string{})).Spread(t.Context(), "v-9", 32, noon)

	require.ErrorIs(t, err, clicks.ErrNoVirus)
}

func TestAVirusTakesForItsPlayerWhoeverAsks(t *testing.T) {
	tiles := claimable(t, map[uint32]string{4: "de", 5: "de"})
	started(t, tiles, 5, 32)

	_, _, err := contagion(tiles).Spread(context.WithoutCancel(t.Context()), "v-1", 32, noon)
	require.NoError(t, err)

	tile, err := tileHeld(tiles, 4)
	require.NoError(t, err)
	assert.Equal(t, carrier, tile.Holder)
}

func tileHeld(tiles *inmemory_tile_storage.Storage, id uint32) (clicks.TileSnapshot, error) {
	var held clicks.TileSnapshot
	err := tiles.Claim(context.Background(), func(locked clicks.Claimable) error {
		tile, err := locked.Tile(id)
		held = tile.TakeSnapshot()
		return err //nolint:wrapcheck // the test reads the storage's own error.
	})
	return held, err //nolint:wrapcheck // the test reads the storage's own error.
}
