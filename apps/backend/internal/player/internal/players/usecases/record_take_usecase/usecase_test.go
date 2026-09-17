package record_take_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_take_usecase"
)

var at = time.Date(2026, 9, 17, 23, 59, 0, 0, time.UTC)

func TestEachTakeIsCountedOnTheAccountsStats(t *testing.T) {
	store := inmemory_player_store.New()
	useCase := record_take_usecase.New(store)

	require.NoError(t, useCase.Execute(t.Context(), record_take_usecase.In{Account: players.AccountID{15: 1}, At: at}))
	require.NoError(t, useCase.Execute(t.Context(), record_take_usecase.In{Account: players.AccountID{15: 1}, At: at.Add(2 * time.Minute)}))

	stats, err := store.Stats(t.Context(), players.AccountID{15: 1})
	require.NoError(t, err)
	assert.Equal(t, players.Stats{
		Account: players.AccountID{15: 1}, TilesTaken: 2, StreakCurrent: 2, StreakBest: 2, StreakLastDay: players.DayOf(at).Following(),
	}, stats)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_player_store.New()
	store.FailWith(errors.New("postgres is down"))

	err := record_take_usecase.New(store).Execute(t.Context(), record_take_usecase.In{Account: players.AccountID{15: 1}, At: at})

	assert.Error(t, err)
}
