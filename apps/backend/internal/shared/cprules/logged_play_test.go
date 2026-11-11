package cprules_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

type stubPlaying struct {
	taken snapshot
	took  bool
	err   error
}

func (s stubPlaying) Execute(context.Context) (snapshot, bool, error) {
	return s.taken, s.took, s.err
}

func (s stubPlaying) Play(context.Context, cprules.Choice) (snapshot, bool, error) {
	return s.taken, s.took, s.err
}

var says = cprules.PlayLog{Taken: "offer rules in force", Failed: "failed to play the offer rules"}

func logged(inner stubPlaying, logs *bytes.Buffer) *cprules.LoggedPlay[*planetv1.Rules] {
	return cprules.NewLoggedPlay(inner, slog.New(slog.NewTextHandler(logs, nil)), says)
}

func TestAPlayThatTookNothingLogsNothing(t *testing.T) {
	var logs bytes.Buffer

	_, took, err := logged(stubPlaying{taken: seeded()}, &logs).Execute(t.Context())

	require.NoError(t, err)
	assert.False(t, took)
	assert.Empty(t, logs.String())
}

func TestTheSetTakenIsLoggedWithEveryValueItPlays(t *testing.T) {
	var logs bytes.Buffer

	_, took, err := logged(stubPlaying{taken: named("rush", quicker), took: true}, &logs).Execute(t.Context())

	require.NoError(t, err)
	assert.True(t, took)
	assert.Contains(t, logs.String(), `level=WARN msg="offer rules in force" set=rush rules=`)
	assert.Contains(t, logs.String(), `\"tempo\":3`)
	assert.Contains(t, logs.String(), `\"burst\":60`)
}

func TestAFailureIsLoggedButNotOneTheShutdownCut(t *testing.T) {
	var logs bytes.Buffer
	cause := errors.New("postgres is down")
	play := logged(stubPlaying{err: cause}, &logs)

	_, _, err := play.Execute(t.Context())
	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs.String(), `level=ERROR msg="failed to play the offer rules" error="postgres is down"`)

	logs.Reset()
	cut, cancel := context.WithCancel(t.Context())
	cancel()
	_, _, err = play.Execute(cut)
	require.ErrorIs(t, err, cause)
	assert.Empty(t, logs.String())
}

func TestAChoiceRefusedIsLogged(t *testing.T) {
	var logs bytes.Buffer

	_, _, err := logged(stubPlaying{taken: seeded(), err: cprules.ErrUnknownSet}, &logs).
		Play(t.Context(), cprules.Choice{Set: "rush"})

	require.ErrorIs(t, err, cprules.ErrUnknownSet)
	assert.Contains(t, logs.String(), `level=ERROR msg="failed to play the offer rules" error="no rule set has this name"`)
}

func TestAChoiceTakenIsLogged(t *testing.T) {
	var logs bytes.Buffer

	_, _, err := logged(stubPlaying{taken: named("rush", asIs), took: true}, &logs).
		Play(t.Context(), cprules.Choice{Set: "rush"})

	require.NoError(t, err)
	assert.Contains(t, logs.String(), `level=WARN msg="offer rules in force" set=rush`)
}
