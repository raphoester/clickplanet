package log_refusals_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/operatorguard/log_refusals"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/operatorguard/operator_query"
)

type stubOperators struct {
	account string
	err     error
}

func (s stubOperators) Operator(context.Context, string) (string, error) {
	return s.account, s.err
}

func TestARefusedCallerIsLoggedWithoutItsCookie(t *testing.T) {
	var logs bytes.Buffer
	cause := fmt.Errorf("%w: account 0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10", operator_query.ErrNotAnOperator)
	operators := log_refusals.New(stubOperators{err: cause}, slog.New(slog.NewTextHandler(&logs, nil)))

	_, err := operators.Operator(t.Context(), "cp_sid=the-secret-token")

	require.ErrorIs(t, err, operator_query.ErrNotAnOperator)
	assert.Contains(t, logs.String(), `level=WARN msg="operator refused"`)
	assert.Contains(t, logs.String(), "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10")
	assert.NotContains(t, logs.String(), "the-secret-token")
}

func TestAnAdmittedOperatorIsNotLoggedHere(t *testing.T) {
	var logs bytes.Buffer
	operators := log_refusals.New(stubOperators{account: "the operator"}, slog.New(slog.NewTextHandler(&logs, nil)))

	account, err := operators.Operator(t.Context(), "cp_sid=token")

	require.NoError(t, err)
	assert.Equal(t, "the operator", account)
	assert.Empty(t, logs.String(), "the operator listener logs the call itself, once it is done")
}
