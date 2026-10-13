package fetch_replay_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_replay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now      = time.Date(2026, 10, 12, 12, 17, 0, 0, time.UTC)
	halfADay = replays.Window{Since: now.Add(-12 * time.Hour), Until: now}
	answered = replays.Replay{Document: []byte(`{"events":[{}]}`), Events: 1, Window: halfADay}
)

func fetching(asked replays.Request) (*replays.ScriptedSource, *replays.MemoryStore, *fetch_replay_usecase.UseCase) {
	source := replays.NewScriptedSource(answered)
	store := &replays.MemoryStore{}
	return source, store, fetch_replay_usecase.New(asked, cptime.NewFixedClock(now), source, store)
}

func TestTheLastHoursAreAskedOfTheBackendAndKept(t *testing.T) {
	source, store, usecase := fetching(replays.Request{Span: 12 * time.Hour})

	replay, err := usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, answered, replay)
	assert.Equal(t, []replays.Window{halfADay}, source.Asked)
	assert.Equal(t, []replays.Replay{answered}, store.Saved)
}

func TestAReplayThatSpansNoTimeAsksTheBackendNothing(t *testing.T) {
	source, store, usecase := fetching(replays.Request{})

	_, err := usecase.Execute(t.Context())

	require.ErrorIs(t, err, replays.ErrInvalidRequest)
	assert.Empty(t, source.Asked)
	assert.Empty(t, store.Saved)
}

func TestABackendThatDoesNotAnswerKeepsNothing(t *testing.T) {
	unreachable := errors.New("ssh failed")
	source, store, usecase := fetching(replays.Request{Span: time.Hour})
	source.FailWith(unreachable)

	_, err := usecase.Execute(t.Context())

	require.ErrorIs(t, err, unreachable)
	assert.Empty(t, store.Saved)
}

func TestAReplayThatCannotBeKeptFailsTheFetch(t *testing.T) {
	full := errors.New("the disk is full")
	_, store, usecase := fetching(replays.Request{Span: time.Hour})
	store.FailWith(full)

	_, err := usecase.Execute(t.Context())

	require.ErrorIs(t, err, full)
}
