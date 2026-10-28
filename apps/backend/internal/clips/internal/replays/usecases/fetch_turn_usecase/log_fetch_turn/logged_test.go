package log_fetch_turn_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_turn_usecase/log_fetch_turn"
)

type stubExecutor struct {
	turn replays.Turn
	err  error
}

func (s stubExecutor) Execute(context.Context) (replays.Turn, error) {
	return s.turn, s.err
}

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := log_fetch_turn.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())
	return logs.String(), err
}

func TestASavedTurnIsLoggedWithItsDebriefAndItsFinale(t *testing.T) {
	logs, err := run(t, stubExecutor{turn: replays.Turn{
		Document: []byte("{}"), Debriefed: "1.5", EndedAt: time.Date(2026, 10, 17, 21, 0, 0, 0, time.UTC),
		FinaleAt: time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC),
	}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=INFO msg="saved the turn" debriefed=1.5 ended=2026-10-17T21:00:00.000Z finale=2026-10-31T21:00:00.000Z`)
	assert.NotContains(t, logs, "{}")
}

func TestATurnWithNothingScoredSaysSo(t *testing.T) {
	logs, err := run(t, stubExecutor{turn: replays.Turn{Document: []byte("{}")}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=INFO msg="saved the turn: nothing scored yet"`)
}

func TestATurnLeftOutIsAWarningThatSaysWhy(t *testing.T) {
	unserved := errors.New("ssh failed: exit status 1: wget: server returned error: HTTP/1.1 404 Not Found")

	logs, err := run(t, stubExecutor{err: unserved})

	require.ErrorIs(t, err, unserved)
	assert.Contains(t, logs, `level=WARN msg="left the turn out: no round of the season is told, no country called up"`)
	assert.Contains(t, logs, "404 Not Found")
}
