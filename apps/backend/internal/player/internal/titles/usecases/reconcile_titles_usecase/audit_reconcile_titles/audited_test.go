package audit_reconcile_titles_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/reconcile_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/reconcile_titles_usecase/audit_reconcile_titles"
)

type stubExecutor struct {
	reconciled reconcile_titles_usecase.Reconciled
	err        error
}

func (s stubExecutor) Execute(context.Context) (reconcile_titles_usecase.Reconciled, error) {
	return s.reconciled, s.err
}

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := audit_reconcile_titles.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())
	return logs.String(), err
}

func TestEveryReconciliationIsLoggedWithWhatItChanged(t *testing.T) {
	logs, err := run(t, stubExecutor{reconciled: reconcile_titles_usecase.Reconciled{Granted: 42, Revoked: 7}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin title reconciliation" granted=42 revoked=7`)

	logs, err = run(t, stubExecutor{})
	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin title reconciliation" granted=0 revoked=0`, "one that changed nothing is logged too")
}

func TestAFailureIsLoggedWithWhatWasDoneBeforeIt(t *testing.T) {
	cause := errors.New("postgres is down")

	logs, err := run(t, stubExecutor{reconciled: reconcile_titles_usecase.Reconciled{Granted: 500}, err: cause})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=WARN msg="admin title reconciliation failed" granted=500 revoked=0`)
}
