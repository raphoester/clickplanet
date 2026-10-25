package bomb_landed_subscriber_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/bomb_landed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

func subscriber() (bomb_landed_subscriber.Subscriber, *inmemory_player_store.Store) {
	store := inmemory_player_store.New()
	return bomb_landed_subscriber.New(record_stats_usecase.New(store, cpbootstrap.NewRecordedEvents())), store
}

func statsOf(t *testing.T, store *inmemory_player_store.Store) players.StatsSnapshot {
	t.Helper()

	id, err := players.AccountIDOf(account)
	require.NoError(t, err)
	stats, err := store.Stats(t.Context(), id)
	require.NoError(t, err)
	return stats.TakeSnapshot()
}

func TestABombCountsOnTheStatsOfTheAccountThatDroppedIt(t *testing.T) {
	handler, store := subscriber()

	require.NoError(t, handler.Handle(t.Context(), &planetv1.BombLanded{AccountId: account, Country: "fr", TileId: 42}))

	assert.Equal(t, uint64(1), statsOf(t, store).BombsDropped)
}

func TestABombNoAccountDroppedCountsForNobody(t *testing.T) {
	handler, store := subscriber()

	require.NoError(t, handler.Handle(t.Context(), &planetv1.BombLanded{Country: "fr", TileId: 42}))

	all, err := store.StatsAfter(t.Context(), players.AccountID{}, 10)
	require.NoError(t, err)
	assert.Empty(t, all)
}

func TestAnAccountThatIsNoAccountIsRefused(t *testing.T) {
	handler, _ := subscriber()

	assert.ErrorIs(t, handler.Handle(t.Context(), &planetv1.BombLanded{AccountId: "nope"}), players.ErrInvalidAccount)
}
