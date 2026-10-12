package log_verifier_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/access"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/access/log_verifier"
)

type stubVerifier struct {
	caller access.Caller
	err    error
}

func (s stubVerifier) Caller(context.Context, string) (access.Caller, error) {
	return s.caller, s.err
}

func TestACallerLetInIsNotLogged(t *testing.T) {
	var logs bytes.Buffer
	verifier := log_verifier.New(stubVerifier{caller: "claude-cloud.access"}, slog.New(slog.NewTextHandler(&logs, nil)))

	caller, err := verifier.Caller(t.Context(), "signed")

	require.NoError(t, err)
	assert.Equal(t, access.Caller("claude-cloud.access"), caller)
	assert.Empty(t, logs.String())
}

func TestACallerRefusedIsLoggedWithTheReasonAndNotItsAssertion(t *testing.T) {
	var logs bytes.Buffer
	expired := errors.New("token is expired")
	verifier := log_verifier.New(stubVerifier{err: expired}, slog.New(slog.NewTextHandler(&logs, nil)))

	_, err := verifier.Caller(t.Context(), "a-signed-assertion")

	require.ErrorIs(t, err, expired)
	assert.Contains(t, logs.String(), "ops refused a caller")
	assert.Contains(t, logs.String(), "token is expired")
	assert.NotContains(t, logs.String(), "a-signed-assertion")
}
