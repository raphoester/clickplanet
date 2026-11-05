package log_fetch_round_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_round_usecase/log_fetch_round"
)

type stubExecutor struct {
	round replays.Round
	err   error
}

func (s stubExecutor) Execute(context.Context, replays.Turn) (replays.Round, error) {
	return s.round, s.err
}

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := log_fetch_round.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), replays.Turn{})
	return logs.String(), err
}

func TestASavedRoundIsLoggedWithItsID(t *testing.T) {
	logs, err := run(t, stubExecutor{round: replays.Round{Document: []byte("{}"), ID: "season-1/day-5"}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=INFO msg="saved the round" round=season-1/day-5`)
	assert.NotContains(t, logs, "{}")
}

func TestNoRoundDebriefedSaysSo(t *testing.T) {
	logs, err := run(t, stubExecutor{round: replays.NoRound})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=INFO msg="saved the round: none was debriefed"`)
}

func TestARoundLeftOutIsAWarningThatSaysWhy(t *testing.T) {
	unserved := errors.New("ssh failed: exit status 1: wget: server returned error: HTTP/1.1 404 Not Found")

	logs, err := run(t, stubExecutor{err: unserved})

	require.ErrorIs(t, err, unserved)
	assert.Contains(t, logs, `level=WARN msg="left the round out: no round of the season is told"`)
	assert.Contains(t, logs, "404 Not Found")
}
