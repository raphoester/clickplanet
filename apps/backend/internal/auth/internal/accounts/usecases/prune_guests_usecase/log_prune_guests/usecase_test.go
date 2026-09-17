package log_prune_guests_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/prune_guests_usecase/log_prune_guests"
)

type stubExecutor struct {
	pruned int
	err    error
}

func (s stubExecutor) Execute(context.Context) (int, error) {
	return s.pruned, s.err
}

func run(t *testing.T, ctx context.Context, inner stubExecutor) (string, int, error) {
	t.Helper()

	var logs bytes.Buffer
	pruned, err := log_prune_guests.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(ctx)
	return logs.String(), pruned, err
}

func TestAPruneThatDeletedSomethingIsLoggedAndPassedOn(t *testing.T) {
	logs, pruned, err := run(t, t.Context(), stubExecutor{pruned: 3})

	require.NoError(t, err)
	assert.Equal(t, 3, pruned)
	assert.Contains(t, logs, "level=INFO msg=\"pruned idle guests\" pruned=3")
}

func TestAPruneThatDeletedNothingLogsNothing(t *testing.T) {
	logs, _, err := run(t, t.Context(), stubExecutor{})

	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestAFailureIsLoggedAndPassedOn(t *testing.T) {
	failure := errors.New("postgres is down")

	logs, pruned, err := run(t, t.Context(), stubExecutor{pruned: 1000, err: failure})

	require.ErrorIs(t, err, failure)
	assert.Equal(t, 1000, pruned)
	assert.Contains(t, logs, "level=ERROR msg=\"failed to prune the idle guests\" pruned=1000 error=\"postgres is down\"")
}

func TestAFailureWhileStoppingIsNotLogged(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	logs, _, err := run(t, ctx, stubExecutor{err: context.Canceled})

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, logs)
}
