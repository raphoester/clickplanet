package audit_backfill_days_played_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/backfill_days_played_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/backfill_days_played_usecase/audit_backfill_days_played"
)

type stubExecutor struct {
	backfilled backfill_days_played_usecase.Backfilled
	err        error
}

func (s stubExecutor) Execute(context.Context, bool) (backfill_days_played_usecase.Backfilled, error) {
	return s.backfilled, s.err
}

func run(t *testing.T, inner stubExecutor, dryRun bool) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := audit_backfill_days_played.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), dryRun)
	return logs.String(), err
}

func TestEveryBackfillIsLoggedWithWhatItChanged(t *testing.T) {
	logs, err := run(t, stubExecutor{backfilled: backfill_days_played_usecase.Backfilled{Accounts: 42, Changed: 7}}, false)

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin days played backfill" dry_run=false accounts=42 changed=7`)

	logs, err = run(t, stubExecutor{}, true)
	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin days played backfill" dry_run=true accounts=0 changed=0`)
}

func TestAFailureIsLoggedWithWhatWasDoneBeforeIt(t *testing.T) {
	cause := errors.New("planet is down")

	logs, err := run(t, stubExecutor{backfilled: backfill_days_played_usecase.Backfilled{Accounts: 500}, err: cause}, false)

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=WARN msg="admin days played backfill failed" dry_run=false accounts=500 changed=0`)
}
