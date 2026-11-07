package tiles_taken_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/inmemory_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/record_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/tiles_taken_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

const (
	ada   = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	grace = "6f1c2f0e-3a8b-4c5d-9e7f-1a2b3c4d5e6f"
)

var at = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func takes(account string, position uint64, from, to uint32) *planetv1.TilesTaken {
	tiles := make([]*planetv1.TakenTile, 0, to-from)
	for index := from; index < to; index++ {
		tiles = append(tiles, &planetv1.TakenTile{TileId: 40 + index, Index: index})
	}
	return &planetv1.TilesTaken{AccountId: account, Country: "fr", TakenAt: timestamppb.New(at), Position: position, Tiles: tiles}
}

func statsOf(t *testing.T, store *inmemory_stats_store.Store, account string) stats.Stats {
	t.Helper()

	id, err := accounts.AccountIDOf(account)
	require.NoError(t, err)
	counted, err := store.Stats(t.Context(), id)
	require.NoError(t, err)
	return counted
}

func TestATakeIsCountedAtTheTimeItWasTaken(t *testing.T) {
	store := inmemory_stats_store.New()

	err := tiles_taken_subscriber.New(record_stats_usecase.New(store, cpredis.NewRecordedEvents())).
		HandleBatch(t.Context(), []*planetv1.TilesTaken{takes(ada, 0, 0, 1)})

	require.NoError(t, err)
	assert.Equal(t, uint64(1), statsOf(t, store, ada).TilesTaken())
	assert.Equal(t, stats.DayOf(at), statsOf(t, store, ada).Streak().LastDay())
}

func TestAnEnclosureToldInPartsAndToldTwiceCountsEachTileOnce(t *testing.T) {
	store := inmemory_stats_store.New()
	events := cpredis.NewRecordedEvents()
	handler := tiles_taken_subscriber.New(record_stats_usecase.New(store, events))
	enclosure := []*planetv1.TilesTaken{takes(ada, 9, 0, 3), takes(ada, 9, 3, 7)}

	for range 2 {
		require.NoError(t, handler.HandleBatch(t.Context(), enclosure))
	}
	require.NoError(t, handler.HandleBatch(t.Context(), enclosure[1:]), "a failed batch is handled again one event at a time")

	assert.Equal(t, uint64(7), statsOf(t, store, ada).TilesTaken(), "one ledger event in two parts, seven takes")
}

func TestEachAccountOfABatchIsCountedOnceWithItsOwnTakes(t *testing.T) {
	store := inmemory_stats_store.New()
	events := cpredis.NewRecordedEvents()

	err := tiles_taken_subscriber.New(record_stats_usecase.New(store, events)).HandleBatch(t.Context(), []*planetv1.TilesTaken{
		takes(ada, 1, 0, 1), takes(grace, 2, 0, 5), takes(ada, 3, 0, 2),
	})

	require.NoError(t, err)
	assert.Equal(t, uint64(3), statsOf(t, store, ada).TilesTaken())
	assert.Equal(t, uint64(5), statsOf(t, store, grace).TilesTaken())
	assert.Len(t, events.Published(), 2, "one StatsChanged per account, not per take")
}

func TestABatchWithAnEventWithNoAccountOrNoTimeIsRefused(t *testing.T) {
	subscriber := tiles_taken_subscriber.New(record_stats_usecase.New(inmemory_stats_store.New(), cpredis.NewRecordedEvents()))

	require.ErrorIs(t, subscriber.HandleBatch(t.Context(), []*planetv1.TilesTaken{takes(ada, 0, 0, 1), takes("", 1, 0, 1)}),
		accounts.ErrInvalidAccount)
	noTime := takes(ada, 2, 0, 1)
	noTime.TakenAt = nil
	assert.Error(t, subscriber.HandleBatch(t.Context(), []*planetv1.TilesTaken{noTime}))
}
