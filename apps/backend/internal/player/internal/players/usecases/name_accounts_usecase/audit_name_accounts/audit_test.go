package audit_name_accounts_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/name_accounts_usecase/audit_name_accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubExecutor struct {
	named int
	err   error
}

func (s stubExecutor) Execute(context.Context) (int, error) {
	return s.named, s.err
}

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := audit_name_accounts.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())
	return logs.String(), err
}

func TestEveryNamingIsLoggedWithHowManyItNamed(t *testing.T) {
	logs, err := run(t, stubExecutor{named: 42})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin account naming" named=42`)
}

func TestAFailureIsLoggedWithWhatWasDoneBeforeIt(t *testing.T) {
	cause := errors.New("postgres is down")

	logs, err := run(t, stubExecutor{named: 500, err: cause})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=WARN msg="admin account naming failed" named=500`)
}

func TestANamingNamesItsOperator(t *testing.T) {
	var logs bytes.Buffer

	_, err := audit_name_accounts.New(stubExecutor{}, slog.New(slog.NewTextHandler(&logs, nil))).
		Execute(cpctx.AddOperatorToContext(t.Context(), "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"))

	require.NoError(t, err)
	assert.Contains(t, logs.String(), "operator=0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10")
}
