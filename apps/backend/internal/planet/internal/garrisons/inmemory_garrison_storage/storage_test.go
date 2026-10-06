package inmemory_garrison_storage_test

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons/inmemory_garrison_storage"
)

const perTile = 3

func loaded(t *testing.T, persistence inmemory_garrison_storage.Persistence) *inmemory_garrison_storage.Storage {
	t.Helper()

	storage := inmemory_garrison_storage.New(inmemory_garrison_storage.Config{}, perTile, persistence,
		slog.New(slog.DiscardHandler))
	require.NoError(t, storage.Load(t.Context()))

	return storage
}

func fresh(t *testing.T) (*inmemory_garrison_storage.Storage, *inmemory_garrison_storage.MemoryPersistence) {
	t.Helper()

	persistence := inmemory_garrison_storage.NewMemoryPersistence()

	return loaded(t, persistence), persistence
}

func TestEachDefenderStandsOnTheTileForItsFlag(t *testing.T) {
	storage, _ := fresh(t)

	storage.Reinforce(7, "fr")
	storage.Reinforce(7, "fr")

	assert.Equal(t, 2, storage.Defenders(7, "fr"))
	assert.Zero(t, storage.Defenders(7, "de"), "defenders stand for the flag that placed them")
	assert.Zero(t, storage.Defenders(8, "fr"))
}

func TestATileHoldsSoManyDefendersAndNoMore(t *testing.T) {
	storage, _ := fresh(t)

	for range perTile + 2 {
		storage.Reinforce(7, "fr")
	}

	assert.Equal(t, perTile, storage.Defenders(7, "fr"))
	assert.True(t, storage.Full(7, "fr"))
	assert.False(t, storage.Full(7, "de"), "another flag would start a garrison of its own")
}

func TestAStrikeTakesOneDefenderAndTheLastOneLeavesNothing(t *testing.T) {
	storage, _ := fresh(t)
	storage.Reinforce(7, "fr")
	storage.Reinforce(7, "fr")

	require.True(t, storage.Strike(7, "fr"))
	require.True(t, storage.Strike(7, "fr"))
	assert.False(t, storage.Strike(7, "fr"), "no defender is left to take the click")
	assert.Empty(t, storage.Garrisons())
}

func TestAGarrisonOfAFlagThatLostTheTileDefendsNothing(t *testing.T) {
	storage, persistence := fresh(t)
	storage.Reinforce(7, "fr")

	assert.False(t, storage.Strike(7, "de"), "the tile is de's now, and nothing stands for de")
	assert.Zero(t, storage.Defenders(7, "fr"), "a strike over a garrison that stands for nobody removes it")

	require.NoError(t, storage.Flush(t.Context()))
	assert.Empty(t, persistence.Stored())
}

func TestEveryChangeOfCountIsPublished(t *testing.T) {
	storage, _ := fresh(t)
	updates, err := storage.Subscribe(t.Context())
	require.NoError(t, err)

	storage.Reinforce(7, "fr")
	storage.Strike(7, "fr")
	storage.Strike(7, "fr")

	assert.Equal(t, garrisons.Garrison{Tile: 7, Country: "fr", Defenders: 1}, <-updates)
	assert.Equal(t, garrisons.Garrison{Tile: 7, Country: "fr"}, <-updates, "none left is said too")
	assert.Empty(t, updates, "a click that hit no defender changes nothing")
}

func TestGarrisonsListsEveryTileHeld(t *testing.T) {
	storage, _ := fresh(t)
	storage.Reinforce(9, "de")
	storage.Reinforce(3, "fr")

	assert.Equal(t, []garrisons.Garrison{
		{Tile: 3, Country: "fr", Defenders: 1},
		{Tile: 9, Country: "de", Defenders: 1},
	}, storage.Garrisons())
}

func TestGarrisonsSurviveARestart(t *testing.T) {
	storage, persistence := fresh(t)
	storage.Reinforce(7, "fr")
	storage.Reinforce(7, "fr")
	storage.Reinforce(8, "de")
	storage.Strike(8, "de")
	require.NoError(t, storage.Flush(t.Context()))

	restarted := loaded(t, persistence)

	assert.Equal(t, 2, restarted.Defenders(7, "fr"))
	assert.Equal(t, []garrisons.Garrison{{Tile: 7, Country: "fr", Defenders: 2}}, restarted.Garrisons())
}

func TestAFailedFlushIsRetried(t *testing.T) {
	storage, persistence := fresh(t)
	storage.Reinforce(7, "fr")

	persistence.FailWith(errors.New("postgres is down"))
	require.Error(t, storage.Flush(t.Context()))

	persistence.Heal()
	require.NoError(t, storage.Flush(t.Context()))
	assert.Equal(t, map[uint32]garrisons.Garrison{7: {Tile: 7, Country: "fr", Defenders: 1}}, persistence.Stored())
}

func TestNothingChangedWritesNothing(t *testing.T) {
	storage, persistence := fresh(t)

	storage.Strike(7, "fr")
	require.NoError(t, storage.Flush(t.Context()))

	assert.Zero(t, persistence.Saves())
}
