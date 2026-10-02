package award_titles_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/award_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	monday = time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC)
	ada    = players.AccountID{15: 1}
)

func takeTiles(t *testing.T, store *inmemory_player_store.Store, tiles int) {
	t.Helper()

	for range tiles {
		require.NoError(t, store.RecordTake(t.Context(), ada, monday))
	}
}

func TestTheTakeThatReachesAThresholdGrantsItsTitle(t *testing.T) {
	store := inmemory_player_store.New()
	useCase := award_titles_usecase.New(store, store, cptime.NewFixedClock(monday))
	takeTiles(t, store, 99)
	require.NoError(t, useCase.Execute(t.Context(), ada))
	require.Empty(t, mustTitles(t, store))

	takeTiles(t, store, 1)
	require.NoError(t, useCase.Execute(t.Context(), ada))

	assert.Equal(t, players.Titles{players.Settler}, mustTitles(t, store))
}

func TestADailyStreakGrantsItsTitle(t *testing.T) {
	store := inmemory_player_store.New()
	for day := range 7 {
		require.NoError(t, store.RecordTake(t.Context(), ada, monday.AddDate(0, 0, day)))
	}

	require.NoError(t, award_titles_usecase.New(store, store, cptime.NewFixedClock(monday)).Execute(t.Context(), ada))

	assert.Equal(t, players.Titles{players.Loyal}, mustTitles(t, store))
}

type refusingGrants struct {
	*inmemory_player_store.Store
}

func (refusingGrants) GrantTitles(context.Context, players.AccountID, players.Titles, time.Time) error {
	return errors.New("nothing was to be granted")
}

func TestNothingIsWrittenWhenEveryEarnedTitleIsHeld(t *testing.T) {
	store := inmemory_player_store.New()
	takeTiles(t, store, 100)
	require.NoError(t, store.GrantTitles(t.Context(), ada, players.Titles{players.Settler}, monday))

	err := award_titles_usecase.New(store, refusingGrants{store}, cptime.NewFixedClock(monday)).Execute(t.Context(), ada)

	assert.NoError(t, err)
}

func TestAnAccountWithNoStatsEarnsNothing(t *testing.T) {
	store := inmemory_player_store.New()

	require.NoError(t, award_titles_usecase.New(store, store, cptime.NewFixedClock(monday)).Execute(t.Context(), ada))

	assert.Empty(t, mustTitles(t, store))
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_player_store.New()
	takeTiles(t, store, 100)
	store.FailWith(errors.New("postgres is down"))

	err := award_titles_usecase.New(store, store, cptime.NewFixedClock(monday)).Execute(t.Context(), ada)

	assert.Error(t, err)
}

func mustTitles(t *testing.T, store *inmemory_player_store.Store) players.Titles {
	t.Helper()

	titles, err := store.Titles(t.Context(), ada)
	require.NoError(t, err)
	return titles
}
