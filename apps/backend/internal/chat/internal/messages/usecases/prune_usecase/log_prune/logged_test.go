package log_prune_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/prune_usecase/log_prune"
)

type stubExecutor struct {
	deleted int64
	err     error
}

func (s stubExecutor) Execute(context.Context) (int64, error) {
	return s.deleted, s.err
}

func run(t *testing.T, ctx context.Context, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := log_prune.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(ctx)
	return logs.String(), err
}

func TestAPruneThatDeletedSomethingIsLogged(t *testing.T) {
	logs, err := run(t, t.Context(), stubExecutor{deleted: 3})

	require.NoError(t, err)
	assert.Contains(t, logs, "level=INFO msg=\"pruned the chat\" deleted=3")
}

func TestAnEmptyPruneSaysNothing(t *testing.T) {
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
