package record_take_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_take_usecase"
)

func TestEachTakeIsCountedOnTheAccountsStats(t *testing.T) {
	storage := inmemory_player_storage.New(inmemory_player_storage.NewMemoryPersistence())
	useCase := record_take_usecase.New(storage)
	at := time.Date(2026, 9, 17, 23, 59, 0, 0, time.UTC)

	useCase.Execute(record_take_usecase.In{Account: players.AccountID{15: 1}, At: at})
	useCase.Execute(record_take_usecase.In{Account: players.AccountID{15: 1}, At: at.Add(2 * time.Minute)})

	stats, ok := storage.Stats(players.AccountID{15: 1})
	require.True(t, ok)
	assert.Equal(t, players.Stats{
		Account: players.AccountID{15: 1}, TilesTaken: 2, StreakCurrent: 2, StreakBest: 2, StreakLastDay: players.DayOf(at).Following(),
	}, stats)
}
