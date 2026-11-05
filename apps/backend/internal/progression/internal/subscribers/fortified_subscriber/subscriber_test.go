package fortified_subscriber_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/inmemory_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/record_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/fortified_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

func subscriber() (fortified_subscriber.Subscriber, *inmemory_stats_store.Store) {
	store := inmemory_stats_store.New()
	return fortified_subscriber.New(record_stats_usecase.New(store, cpredis.NewRecordedEvents())), store
}

func statsOf(t *testing.T, store *inmemory_stats_store.Store) stats.StatsSnapshot {
	t.Helper()

	id, err := accounts.AccountIDOf(account)
	require.NoError(t, err)
	stats, err := store.Stats(t.Context(), id)
	require.NoError(t, err)
	return stats.TakeSnapshot()
}

func TestAFortifyCountsEveryTileOfTheLandmass(t *testing.T) {
	handler, store := subscriber()

	require.NoError(t, handler.Handle(t.Context(), &planetv1.Fortified{AccountId: account, Country: "es", Tiles: 137, Position: 4}))
	require.NoError(t, handler.Handle(t.Context(), &planetv1.Fortified{AccountId: account, Country: "es", Tiles: 1, Position: 9}))
	require.NoError(t, handler.Handle(t.Context(), &planetv1.Fortified{AccountId: account, Country: "es", Tiles: 1, Position: 9}))

	assert.Equal(t, uint64(138), statsOf(t, store).TilesFortified, "a fortify told twice counts once")
}

func TestAFortifyNoAccountMadeCountsForNobody(t *testing.T) {
	handler, store := subscriber()

	require.NoError(t, handler.Handle(t.Context(), &planetv1.Fortified{Country: "es", Tiles: 137}))

	all, err := store.StatsAfter(t.Context(), accounts.AccountID{}, 10)
	require.NoError(t, err)
	assert.Empty(t, all)
}
