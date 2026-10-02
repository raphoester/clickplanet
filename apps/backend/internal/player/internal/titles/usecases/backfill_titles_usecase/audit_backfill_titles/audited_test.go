package audit_backfill_titles_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/backfill_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/backfill_titles_usecase/audit_backfill_titles"
)

type stubExecutor struct {
	backfill backfill_titles_usecase.Backfill
	err      error
}

func (s stubExecutor) Execute(context.Context) (backfill_titles_usecase.Backfill, error) {
	return s.backfill, s.err
}

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := audit_backfill_titles.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())
	return logs.String(), err
}

func TestEveryBackfillIsLoggedWithItsAccounts(t *testing.T) {
	logs, err := run(t, stubExecutor{backfill: backfill_titles_usecase.Backfill{Accounts: 42}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin title backfill" accounts=42`)

	logs, err = run(t, stubExecutor{})
	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin title backfill" accounts=0`, "a backfill that granted nothing is logged too")
}

func TestAFailureIsLoggedWithWhatWasDoneBeforeIt(t *testing.T) {
	cause := errors.New("postgres is down")

	logs, err := run(t, stubExecutor{backfill: backfill_titles_usecase.Backfill{Accounts: 500}, err: cause})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=WARN msg="admin title backfill failed" accounts=500`)
}
