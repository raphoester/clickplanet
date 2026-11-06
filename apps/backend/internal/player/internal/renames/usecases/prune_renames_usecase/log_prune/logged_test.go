package log_prune_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/renames/usecases/prune_renames_usecase/log_prune"
)

type stubExecutor struct {
	deleted int64
	err     error
}

func (s stubExecutor) Execute(context.Context) (int64, error) { return s.deleted, s.err }

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := log_prune.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())
	return logs.String(), err
}

func TestAPruneThatDeletedSomethingIsLogged(t *testing.T) {
	logs, err := run(t, stubExecutor{deleted: 3})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=INFO msg="pruned the renames" deleted=3`)
}

func TestAPruneThatDeletedNothingIsQuiet(t *testing.T) {
	logs, err := run(t, stubExecutor{})

	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestAFailedPruneIsLoggedAsAnError(t *testing.T) {
	cause := errors.New("postgres is down")

	logs, err := run(t, stubExecutor{err: cause})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=ERROR msg="failed to prune the renames, retrying next tick"`)
}
