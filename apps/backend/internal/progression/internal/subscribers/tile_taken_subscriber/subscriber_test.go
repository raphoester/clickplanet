package tile_taken_subscriber_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/tile_taken_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var at = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func TestATakeIsCountedAtTheTimeItWasTaken(t *testing.T) {
	store := inmemory_stats_store.New()

	err := tile_taken_subscriber.New(record_stats_usecase.New(store, cpredis.NewRecordedEvents())).Handle(t.Context(),
		&planetv1.TileTaken{AccountId: account, TileId: 42, Country: "fr", TakenAt: timestamppb.New(at)})

	require.NoError(t, err)
	id, err := accounts.AccountIDOf(account)
	require.NoError(t, err)
	counted, err := store.Stats(t.Context(), id)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), counted.TilesTaken())
	assert.Equal(t, stats.DayOf(at), counted.Streak().LastDay())
}

func TestASpreadCountsEachTileItTookOnce(t *testing.T) {
	store := inmemory_stats_store.New()
	handler := tile_taken_subscriber.New(record_stats_usecase.New(store, cpredis.NewRecordedEvents()))
	spread := make([]*planetv1.TileTaken, 0, 7)
	for index := range uint32(7) {
		spread = append(spread, &planetv1.TileTaken{
			AccountId: account, TileId: 40 + index, Country: "fr", TakenAt: timestamppb.New(at), Position: 9, Index: index,
		})
	}

	for range 2 {
		for _, take := range spread {
			require.NoError(t, handler.Handle(t.Context(), take))
		}
	}

	id, err := accounts.AccountIDOf(account)
	require.NoError(t, err)
	counted, err := store.Stats(t.Context(), id)
	require.NoError(t, err)
	assert.Equal(t, uint64(7), counted.TilesTaken(), "one ledger event, seven takes, told twice")
}

func TestAnEventWithNoAccountOrNoTimeIsRefused(t *testing.T) {
	subscriber := tile_taken_subscriber.New(record_stats_usecase.New(inmemory_stats_store.New(), cpredis.NewRecordedEvents()))

	require.ErrorIs(t, subscriber.Handle(t.Context(), &planetv1.TileTaken{TileId: 42, TakenAt: timestamppb.New(at)}),
		accounts.ErrInvalidAccount)
	assert.Error(t, subscriber.Handle(t.Context(), &planetv1.TileTaken{AccountId: account, TileId: 42}))
}
