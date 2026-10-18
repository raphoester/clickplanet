package fetch_race_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_race_usecase"
)

var answered = replays.Race{
	Document: []byte(`{"race":{"closed":{"season":1,"number":5}}}`), Closed: "1.5", EndedAt: time.Date(2026, 10, 17, 21, 0, 0, 0, time.UTC),
}

func fetching() (*replays.ScriptedSource, *replays.MemoryStore, *fetch_race_usecase.UseCase) {
	source := replays.NewScriptedSource(replays.Replay{})
	source.AnswerRace(answered)
	store := &replays.MemoryStore{}
	return source, store, fetch_race_usecase.New(source, store)
}

func TestTheRaceIsAskedOfTheBackendAndKept(t *testing.T) {
	source, store, usecase := fetching()

	race, err := usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, answered, race)
	assert.Equal(t, 1, source.RacesAsked)
	assert.Equal(t, []replays.Race{answered}, store.SavedRaces)
}

func TestABackendThatDoesNotAnswerTheRaceKeepsNothing(t *testing.T) {
	unserved := errors.New("ssh failed: exit status 1: wget: server returned error: HTTP/1.1 404 Not Found")
	source, store, usecase := fetching()
	source.FailRaceWith(unserved)

	_, err := usecase.Execute(t.Context())

	require.ErrorIs(t, err, unserved)
	assert.Empty(t, store.SavedRaces)
}

func TestARaceThatCannotBeKeptIsAnError(t *testing.T) {
	full := errors.New("the disk is full")
	_, store, usecase := fetching()
	store.FailWith(full)

	_, err := usecase.Execute(t.Context())

	require.ErrorIs(t, err, full)
}
