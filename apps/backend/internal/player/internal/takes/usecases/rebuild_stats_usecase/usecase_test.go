package rebuild_stats_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes/inmemory_take_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes/usecases/rebuild_stats_usecase"
)

func TestARebuildPutsTheStatsBackToTheirStartAndSaysWhere(t *testing.T) {
	ada := players.AccountID{15: 1}
	stats := inmemory_player_store.New()
	store := inmemory_take_store.New(stats)
	require.NoError(t, store.Begin(t.Context(), 7))
	batch, err := takes.BatchOf(7, 8, []takes.Take{takes.TakeOf(7, ada, "fr", time.Now(), false)})
	require.NoError(t, err)
	require.NoError(t, store.Count(t.Context(), batch))

	out, err := rebuild_stats_usecase.New(store).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, rebuild_stats_usecase.Out{From: 7}, out)
	position, err := store.Position(t.Context())
	require.NoError(t, err)
	assert.Equal(t, takes.Position(7), position)
	counted, err := stats.Stats(t.Context(), ada)
	require.NoError(t, err)
	assert.Zero(t, counted.TilesTaken())
}

func TestStatsThatNeverBeganCannotBeRebuilt(t *testing.T) {
	_, err := rebuild_stats_usecase.New(inmemory_take_store.New(inmemory_player_store.New())).Execute(t.Context())

	assert.ErrorIs(t, err, takes.ErrNotStarted)
}

func TestAFailedRebuildIsAnError(t *testing.T) {
	store := inmemory_take_store.New(inmemory_player_store.New())
	failed := errors.New("postgres is down")
	store.FailWith(failed)

	_, err := rebuild_stats_usecase.New(store).Execute(t.Context())

	assert.ErrorIs(t, err, failed)
}
