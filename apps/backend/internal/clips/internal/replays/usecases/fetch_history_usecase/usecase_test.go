package fetch_history_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_history_usecase"
)

var (
	started   = time.Date(2026, 11, 1, 15, 0, 0, 0, time.UTC)
	until     = time.Date(2026, 11, 10, 14, 20, 0, 0, time.UTC)
	running   = replays.Turn{SeasonSince: started}
	season    = replays.Window{Since: started, Until: until}
	seasonMap = replays.History{Document: []byte(`{"history":{"frames":[]}}`), Frames: 224, Window: season}
)

func fetching() (*replays.ScriptedSource, *replays.MemoryStore, *fetch_history_usecase.UseCase) {
	source := replays.NewScriptedSource(replays.Replay{})
	source.AnswerHistory(seasonMap)
	store := &replays.MemoryStore{}
	return source, store, fetch_history_usecase.New(source, store)
}

func TestTheSeasonsMapIsAskedFromItsStartToTheReplaysEndAndKept(t *testing.T) {
	source, store, usecase := fetching()

	history, err := usecase.Execute(t.Context(), running, until)

	require.NoError(t, err)
	assert.Equal(t, seasonMap, history)
	assert.Equal(t, []replays.Window{season}, source.HistoriesAsked)
	assert.Equal(t, []replays.History{seasonMap}, store.SavedHistories)
}

func TestNoSeasonAsksForNoMap(t *testing.T) {
	source, store, usecase := fetching()

	_, err := usecase.Execute(t.Context(), replays.Turn{}, until)

	require.ErrorIs(t, err, replays.ErrNoSeason)
	assert.Empty(t, source.HistoriesAsked)
	assert.Empty(t, store.SavedHistories)
}

func TestABackendThatDoesNotAnswerTheMapKeepsNothing(t *testing.T) {
	unserved := errors.New("ssh failed: exit status 1: wget: server returned error: HTTP/1.1 500 Internal Server Error")
	source, store, usecase := fetching()
	source.FailHistoryWith(unserved)

	_, err := usecase.Execute(t.Context(), running, until)

	require.ErrorIs(t, err, unserved)
	assert.Empty(t, store.SavedHistories)
}
