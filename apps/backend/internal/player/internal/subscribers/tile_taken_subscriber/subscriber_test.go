package tile_taken_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/tile_taken_subscriber"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var at = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func TestATakeIsCountedAtTheTimeItWasTaken(t *testing.T) {
	storage := inmemory_player_storage.New(inmemory_player_storage.NewMemoryPersistence())

	err := tile_taken_subscriber.New(record_take_usecase.New(storage)).Handle(t.Context(),
		&planetv1.TileTaken{AccountId: account, TileId: 42, Country: "fr", TakenAt: timestamppb.New(at)})

	require.NoError(t, err)
	id, err := players.AccountIDOf(account)
	require.NoError(t, err)
	stats, ok := storage.Stats(id)
	require.True(t, ok)
	assert.Equal(t, uint64(1), stats.TilesTaken)
	assert.Equal(t, players.DayOf(at), stats.StreakLastDay)
}

func TestAnEventWithNoAccountOrNoTimeIsRefused(t *testing.T) {
	subscriber := tile_taken_subscriber.New(record_take_usecase.New(inmemory_player_storage.New(inmemory_player_storage.NewMemoryPersistence())))

	require.ErrorIs(t, subscriber.Handle(t.Context(), &planetv1.TileTaken{TileId: 42, TakenAt: timestamppb.New(at)}),
		players.ErrInvalidAccount)
	assert.Error(t, subscriber.Handle(t.Context(), &planetv1.TileTaken{AccountId: account, TileId: 42}))
}
