package log_purge_deletions_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/purge_deletions_usecase/log_purge_deletions"
)

type stubExecutor struct {
	purged int
	err    error
}

func (s stubExecutor) Execute(context.Context) (int, error) {
	return s.purged, s.err
}

func run(t *testing.T, ctx context.Context, inner stubExecutor) (string, int, error) {
	t.Helper()

	var logs bytes.Buffer
	purged, err := log_purge_deletions.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(ctx)
	return logs.String(), purged, err
}

func TestAPurgeThatDeletedSomethingIsLoggedAndPassedOn(t *testing.T) {
	logs, purged, err := run(t, t.Context(), stubExecutor{purged: 3})

	require.NoError(t, err)
	assert.Equal(t, 3, purged)
	assert.Contains(t, logs, "level=INFO msg=\"purged accounts whose deletion was requested\" purged=3")
}

func TestAPurgeThatDeletedNothingLogsNothing(t *testing.T) {
	logs, _, err := run(t, t.Context(), stubExecutor{})

	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestAFailureIsLoggedAndPassedOn(t *testing.T) {
	failure := errors.New("postgres is down")

	logs, purged, err := run(t, t.Context(), stubExecutor{purged: 1000, err: failure})

	require.ErrorIs(t, err, failure)
	assert.Equal(t, 1000, purged)
	assert.Contains(t, logs, "level=ERROR msg=\"failed to purge the accounts whose deletion was requested\" purged=1000 error=\"postgres is down\"")
}

func TestAFailureWhileStoppingIsNotLogged(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	logs, _, err := run(t, ctx, stubExecutor{err: context.Canceled})

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, logs)
}
