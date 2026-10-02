package log_backfill_titles_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/backfill_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/backfill_titles_usecase/log_backfill_titles"
)

type stubExecutor struct {
	backfill backfill_titles_usecase.Backfill
	err      error
}

func (s stubExecutor) Execute(context.Context) (backfill_titles_usecase.Backfill, error) {
	return s.backfill, s.err
}

func run(t *testing.T, ctx context.Context, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := log_backfill_titles.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(ctx)
	return logs.String(), err
}

func TestABackfillIsLoggedWithItsTitlesAndAccounts(t *testing.T) {
	logs, err := run(t, t.Context(), stubExecutor{backfill: backfill_titles_usecase.Backfill{
		Titles: players.TitleIDs{"settler", "loyal"}, Accounts: 42,
	}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=INFO msg="backfilled the titles" titles="[settler loyal]" accounts=42`)
}

func TestNothingToBackfillSaysNothing(t *testing.T) {
	logs, err := run(t, t.Context(), stubExecutor{})

	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestAFailureIsLoggedUnlessTheProcessIsStopping(t *testing.T) {
	cause := errors.New("postgres is down")

	logs, err := run(t, t.Context(), stubExecutor{err: cause})
	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, "level=ERROR")

	stopped, cancel := context.WithCancel(t.Context())
	cancel()
	logs, _ = run(t, stopped, stubExecutor{err: cause})
	assert.Empty(t, logs)
}
