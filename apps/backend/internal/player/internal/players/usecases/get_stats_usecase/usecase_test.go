package get_stats_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var monday = time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC)

func TestAnAccountThatNeverTookATileHasEmptyStats(t *testing.T) {
	stats, err := get_stats_usecase.New(inmemory_player_store.New(), cptime.NewFixedClock(monday)).Execute(t.Context(), players.AccountID{15: 1})

	require.NoError(t, err)
	assert.Equal(t, players.Stats{Account: players.AccountID{15: 1}}, stats)
}

func TestTheStreakIsReadAsOfTodayInUTC(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.RecordTake(t.Context(), players.AccountID{15: 1}, monday))
	require.NoError(t, store.RecordTake(t.Context(), players.AccountID{15: 1}, monday.Add(24*time.Hour)))
	clock := cptime.NewFixedClock(monday.Add(48 * time.Hour))
	useCase := get_stats_usecase.New(store, clock)

	stats, err := useCase.Execute(t.Context(), players.AccountID{15: 1})
	require.NoError(t, err)
	assert.Equal(t, uint32(2), stats.StreakCurrent, "Wednesday can still extend it")

	clock.Advance(24 * time.Hour)
	stats, err = useCase.Execute(t.Context(), players.AccountID{15: 1})
	require.NoError(t, err)
	assert.Equal(t, uint32(0), stats.StreakCurrent, "Thursday: Wednesday went by with no take")
	assert.Equal(t, uint32(2), stats.StreakBest)
	assert.Equal(t, uint64(2), stats.TilesTaken)
}

func TestAStoreFailureIsNotEmptyStats(t *testing.T) {
	store := inmemory_player_store.New()
	store.FailWith(errors.New("postgres is down"))

	_, err := get_stats_usecase.New(store, cptime.NewFixedClock(monday)).Execute(t.Context(), players.AccountID{15: 1})

	require.Error(t, err)
	assert.NotErrorIs(t, err, players.ErrNoStats)
}
