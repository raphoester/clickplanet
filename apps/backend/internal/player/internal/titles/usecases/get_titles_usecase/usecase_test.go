package get_titles_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/get_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	monday = time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC)
	ada    = players.AccountID{15: 1}
)

func TestTheDashboardMeasuresTheStreakAsOfToday(t *testing.T) {
	stats, held := inmemory_player_store.New(), inmemory_title_store.New()
	require.NoError(t, stats.RecordTake(t.Context(), ada, monday))
	require.NoError(t, stats.RecordTake(t.Context(), ada, monday.Add(24*time.Hour)))
	require.NoError(t, held.Grant(t.Context(), titles.Holdings{ada: {"settler"}}, monday))
	clock := cptime.NewFixedClock(monday.Add(24 * time.Hour))
	useCase := get_titles_usecase.New(stats, titles.NewBook(held, titles.NewCatalog()), clock)

	dashboard, err := useCase.Execute(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, "settler", string(dashboard.Worn.Title.ID()))
	assert.Equal(t, uint64(2), dashboard.Tracks[0].Progress, "tiles taken")
	assert.Equal(t, uint64(2), dashboard.Tracks[1].Progress, "the streak on its second day")

	clock.Advance(72 * time.Hour)
	dashboard, err = useCase.Execute(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), dashboard.Tracks[1].Progress, "a streak broken since reads zero")
}

func TestAnAccountWithNoStatsStartsEveryTrackAtZero(t *testing.T) {
	useCase := get_titles_usecase.New(inmemory_player_store.New(), titles.NewBook(inmemory_title_store.New(), titles.NewCatalog()),
		cptime.NewFixedClock(monday))

	dashboard, err := useCase.Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.True(t, dashboard.Worn.Empty())
	assert.Empty(t, dashboard.Wearable)
	for _, track := range dashboard.Tracks {
		assert.Zero(t, track.Progress, track.ID)
	}
}

func TestAStoreFailureIsAnError(t *testing.T) {
	stats := inmemory_player_store.New()
	stats.FailWith(errors.New("postgres is down"))
	useCase := get_titles_usecase.New(stats, titles.NewBook(inmemory_title_store.New(), titles.NewCatalog()), cptime.NewFixedClock(monday))

	_, err := useCase.Execute(t.Context(), ada)

	assert.Error(t, err)
}
