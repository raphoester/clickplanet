package get_stats_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var monday = time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC)

func TestAnAccountThatNeverTookATileHasEmptyStats(t *testing.T) {
	storage := inmemory_player_storage.New(inmemory_player_storage.NewMemoryPersistence())

	stats := get_stats_usecase.New(storage, cptime.NewFixedClock(monday)).Execute(players.AccountID{15: 1})

	assert.Equal(t, players.Stats{Account: players.AccountID{15: 1}}, stats)
}

func TestTheStreakIsReadAsOfTodayInUTC(t *testing.T) {
	storage := inmemory_player_storage.New(inmemory_player_storage.NewMemoryPersistence())
	storage.RecordTake(players.AccountID{15: 1}, monday)
	storage.RecordTake(players.AccountID{15: 1}, monday.Add(24*time.Hour))
	clock := cptime.NewFixedClock(monday.Add(48 * time.Hour))
	useCase := get_stats_usecase.New(storage, clock)

	assert.Equal(t, uint32(2), useCase.Execute(players.AccountID{15: 1}).StreakCurrent, "Wednesday can still extend it")

	clock.Advance(24 * time.Hour)
	stats := useCase.Execute(players.AccountID{15: 1})
	assert.Equal(t, uint32(0), stats.StreakCurrent, "Thursday: Wednesday went by with no take")
	assert.Equal(t, uint32(2), stats.StreakBest)
	assert.Equal(t, uint64(2), stats.TilesTaken)
}
