package log_fetch_season_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_season_usecase/log_fetch_season"
)

type stubExecutor struct {
	season replays.Season
	err    error
}

func (s stubExecutor) Execute(context.Context) (replays.Season, error) {
	return s.season, s.err
}

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := log_fetch_season.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())
	return logs.String(), err
}

func TestASavedSeasonIsLoggedWithItsFinale(t *testing.T) {
	logs, err := run(t, stubExecutor{season: replays.Season{
		Document: []byte("{}"), FinaleStartsAt: time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC),
	}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=INFO msg="saved the season" finale=2026-10-31T21:00:00.000Z`)
	assert.NotContains(t, logs, "{}")
}

func TestNoSeasonOnSaysSo(t *testing.T) {
	logs, err := run(t, stubExecutor{season: replays.Season{Document: []byte("{}")}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=INFO msg="saved the season: none is on"`)
}

func TestASeasonLeftOutIsAWarningThatSaysWhy(t *testing.T) {
	unserved := errors.New("ssh failed: exit status 1: wget: server returned error: HTTP/1.1 502 Bad Gateway")

	logs, err := run(t, stubExecutor{err: unserved})

	require.ErrorIs(t, err, unserved)
	assert.Contains(t, logs, `level=WARN msg="left the season out: no country is called up"`)
	assert.Contains(t, logs, "502 Bad Gateway")
}
