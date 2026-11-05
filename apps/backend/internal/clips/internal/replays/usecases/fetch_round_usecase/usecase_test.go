package fetch_round_usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_round_usecase"
)

var (
	debriefed = replays.Turn{Debriefed: "1.5", Round: "season-1/day-5"}
	answered  = replays.Round{Document: []byte(`{"round":{"roundId":"season-1/day-5"}}`), ID: "season-1/day-5"}
)

func fetching() (*replays.ScriptedSource, *replays.MemoryStore, *fetch_round_usecase.UseCase) {
	source := replays.NewScriptedSource(replays.Replay{})
	source.AnswerRound(answered)
	store := &replays.MemoryStore{}
	return source, store, fetch_round_usecase.New(source, store)
}

func TestTheDebriefedRoundIsAskedOfTheBackendAndKept(t *testing.T) {
	source, store, usecase := fetching()

	round, err := usecase.Execute(t.Context(), debriefed)

	require.NoError(t, err)
	assert.Equal(t, answered, round)
	assert.Equal(t, []replays.RoundID{"season-1/day-5"}, source.RoundsAsked)
	assert.Equal(t, []replays.Round{answered}, store.SavedRounds)
}

func TestATurnWithNoDebriefKeepsAnEmptyRoundAndAsksNothing(t *testing.T) {
	source, store, usecase := fetching()

	round, err := usecase.Execute(t.Context(), replays.Turn{})

	require.NoError(t, err)
	assert.Equal(t, replays.NoRound, round)
	assert.Empty(t, source.RoundsAsked)
	assert.Equal(t, []replays.Round{replays.NoRound}, store.SavedRounds)
}

func TestABackendThatDoesNotAnswerTheRoundKeepsNothing(t *testing.T) {
	unserved := errors.New("ssh failed: exit status 1: wget: server returned error: HTTP/1.1 404 Not Found")
	source, store, usecase := fetching()
	source.FailRoundWith(unserved)

	_, err := usecase.Execute(t.Context(), debriefed)

	require.ErrorIs(t, err, unserved)
	assert.Empty(t, store.SavedRounds)
}

func TestARoundThatCannotBeKeptIsAnError(t *testing.T) {
	full := errors.New("the disk is full")
	_, store, usecase := fetching()
	store.FailWith(full)

	_, err := usecase.Execute(t.Context(), debriefed)

	require.ErrorIs(t, err, full)
}
