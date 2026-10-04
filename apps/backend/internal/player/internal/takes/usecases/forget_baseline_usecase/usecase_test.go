package forget_baseline_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes/inmemory_take_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes/usecases/forget_baseline_usecase"
)

func TestADeletedAccountIsNotPutBackByARebuild(t *testing.T) {
	ada := players.AccountID{15: 1}
	stats := inmemory_player_store.New()
	require.NoError(t, stats.RecordTake(t.Context(), ada, time.Now()))
	store := inmemory_take_store.New(stats)
	require.NoError(t, store.Begin(t.Context(), 0))

	require.NoError(t, forget_baseline_usecase.New(store).Execute(t.Context(), ada))

	_, err := store.Rewind(t.Context())
	require.NoError(t, err)
	counted, err := stats.Stats(t.Context(), ada)
	require.NoError(t, err)
	assert.Zero(t, counted.TilesTaken())
}

func TestAFailureIsAnError(t *testing.T) {
	store := inmemory_take_store.New(inmemory_player_store.New())
	failed := errors.New("postgres is down")
	store.FailWith(failed)

	assert.ErrorIs(t, forget_baseline_usecase.New(store).Execute(t.Context(), players.AccountID{15: 1}), failed)
}
