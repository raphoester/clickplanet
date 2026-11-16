package audit_reconcile_levels_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/usecases/reconcile_levels_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/usecases/reconcile_levels_usecase/audit_reconcile_levels"
)

type stubExecutor struct {
	reconciled reconcile_levels_usecase.Reconciled
	err        error
}

func (s stubExecutor) Execute(context.Context, bool) (reconcile_levels_usecase.Reconciled, error) {
	return s.reconciled, s.err
}

func run(t *testing.T, inner stubExecutor, dryRun bool) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := audit_reconcile_levels.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), dryRun)
	return logs.String(), err
}

func TestEveryReconciliationIsLoggedWithWhatItChanged(t *testing.T) {
	logs, err := run(t, stubExecutor{reconciled: reconcile_levels_usecase.Reconciled{Raised: 42, Lowered: 7, Removed: 1}}, true)

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin level reconciliation" dry_run=true raised=42 lowered=7 removed=1`)

	logs, err = run(t, stubExecutor{}, false)
	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin level reconciliation" dry_run=false raised=0 lowered=0 removed=0`,
		"one that changed nothing is logged too")
}

func TestAFailureIsLoggedWithWhatWasDoneBeforeIt(t *testing.T) {
	cause := errors.New("player is down")

	logs, err := run(t, stubExecutor{reconciled: reconcile_levels_usecase.Reconciled{Raised: 500}, err: cause}, false)

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=WARN msg="admin level reconciliation failed" dry_run=false raised=500 lowered=0 removed=0`)
}
