package fetch_turn_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_turn_usecase"
)

var answered = replays.Turn{
	Document:  []byte(`{"debrief":{"day":{"season":1,"day":5,"roundId":"season-1/day-5"}}}`),
	Debriefed: "1.5", Round: "season-1/day-5", EndedAt: time.Date(2026, 10, 17, 21, 0, 0, 0, time.UTC),
}

func fetching() (*replays.ScriptedSource, *replays.MemoryStore, *fetch_turn_usecase.UseCase) {
	source := replays.NewScriptedSource(replays.Replay{})
	source.AnswerTurn(answered)
	store := &replays.MemoryStore{}
	return source, store, fetch_turn_usecase.New(source, store)
}

func TestTheTurnIsAskedOfTheBackendAndKept(t *testing.T) {
	source, store, usecase := fetching()

	turn, err := usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, answered, turn)
	assert.Equal(t, 1, source.TurnsAsked)
	assert.Equal(t, []replays.Turn{answered}, store.SavedTurns)
}

func TestABackendThatDoesNotAnswerTheTurnKeepsNothing(t *testing.T) {
	unserved := errors.New("ssh failed: exit status 1: wget: server returned error: HTTP/1.1 404 Not Found")
	source, store, usecase := fetching()
	source.FailTurnWith(unserved)

	_, err := usecase.Execute(t.Context())

	require.ErrorIs(t, err, unserved)
	assert.Empty(t, store.SavedTurns)
}

func TestATurnThatCannotBeKeptIsAnError(t *testing.T) {
	full := errors.New("the disk is full")
	_, store, usecase := fetching()
	store.FailWith(full)

	_, err := usecase.Execute(t.Context())

	require.ErrorIs(t, err, full)
}
