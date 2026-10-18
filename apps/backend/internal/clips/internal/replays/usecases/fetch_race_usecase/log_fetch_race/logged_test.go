package log_fetch_race_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_race_usecase/log_fetch_race"
)

type stubExecutor struct {
	race replays.Race
	err  error
}

func (s stubExecutor) Execute(context.Context) (replays.Race, error) {
	return s.race, s.err
}

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := log_fetch_race.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())
	return logs.String(), err
}

func TestASavedRaceIsLoggedWithTheDayClosedLast(t *testing.T) {
	logs, err := run(t, stubExecutor{race: replays.Race{
		Document: []byte("{}"), Closed: "1.5", EndedAt: time.Date(2026, 10, 17, 21, 0, 0, 0, time.UTC),
	}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=INFO msg="saved the race" closed=1.5 ended=2026-10-17T21:00:00.000Z`)
	assert.NotContains(t, logs, "{}")
}

func TestARaceWithNoDayClosedSaysSo(t *testing.T) {
	logs, err := run(t, stubExecutor{race: replays.Race{Document: []byte("{}")}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=INFO msg="saved the race: no day closed yet"`)
}

func TestARaceLeftOutIsAWarningThatSaysWhy(t *testing.T) {
	unserved := errors.New("ssh failed: exit status 1: wget: server returned error: HTTP/1.1 404 Not Found")

	logs, err := run(t, stubExecutor{err: unserved})

	require.ErrorIs(t, err, unserved)
	assert.Contains(t, logs, `level=WARN msg="left the race out: no day of the season is told"`)
	assert.Contains(t, logs, "404 Not Found")
}
