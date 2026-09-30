package log_prune_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity/usecases/prune_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity/usecases/prune_usecase/log_prune"
)

type stubExecutor struct {
	pruned prune_usecase.Pruned
	err    error
}

func (s stubExecutor) Execute(context.Context) (prune_usecase.Pruned, error) {
	return s.pruned, s.err
}

func run(t *testing.T, ctx context.Context, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := log_prune.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(ctx)
	return logs.String(), err
}

func TestAPruneThatDeletedSomethingIsLogged(t *testing.T) {
	logs, err := run(t, t.Context(), stubExecutor{pruned: prune_usecase.Pruned{Expired: 3}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=INFO msg="pruned the activity" expired=3 excess=0`)
}

func TestAPruneThatHitTheCapIsAWarning(t *testing.T) {
	logs, err := run(t, t.Context(), stubExecutor{pruned: prune_usecase.Pruned{Expired: 3, Excess: 2}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="the activity is full, the oldest events went before the retention" expired=3 excess=2`)
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
