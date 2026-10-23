package fetch_season_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_season_usecase"
)

var answered = replays.Season{
	Document:       []byte(`{"season":{"number":0,"finaleStartsAtUnixMs":"1793480400000"}}`),
	FinaleStartsAt: time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC),
}

func fetching() (*replays.ScriptedSource, *replays.MemoryStore, *fetch_season_usecase.UseCase) {
	source := replays.NewScriptedSource(replays.Replay{})
	source.AnswerSeason(answered)
	store := &replays.MemoryStore{}
	return source, store, fetch_season_usecase.New(source, store)
}

func TestTheSeasonIsAskedOfTheBackendAndKept(t *testing.T) {
	source, store, usecase := fetching()

	season, err := usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, answered, season)
	assert.Equal(t, 1, source.SeasonsAsked)
	assert.Equal(t, []replays.Season{answered}, store.SavedSeasons)
}

func TestABackendThatDoesNotAnswerTheSeasonKeepsNothing(t *testing.T) {
	unserved := errors.New("ssh failed: exit status 1: wget: server returned error: HTTP/1.1 502 Bad Gateway")
	source, store, usecase := fetching()
	source.FailSeasonWith(unserved)

	_, err := usecase.Execute(t.Context())

	require.ErrorIs(t, err, unserved)
	assert.Empty(t, store.SavedSeasons)
}

func TestASeasonThatCannotBeKeptIsAnError(t *testing.T) {
	full := errors.New("the disk is full")
	_, store, usecase := fetching()
	store.FailWith(full)

	_, err := usecase.Execute(t.Context())

	require.ErrorIs(t, err, full)
}
