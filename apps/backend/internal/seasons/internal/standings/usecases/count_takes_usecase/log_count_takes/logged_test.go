package log_count_takes_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/count_takes_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/count_takes_usecase/log_count_takes"
)

type stubExecutor struct {
	out count_takes_usecase.Out
	err error
}

func (s stubExecutor) Execute(context.Context) (count_takes_usecase.Out, error) {
	return s.out, s.err
}

func run(t *testing.T, ctx context.Context, inner stubExecutor) (string, count_takes_usecase.Out, error) {
	t.Helper()

	var logs bytes.Buffer
	out, err := log_count_takes.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(ctx)
	return logs.String(), out, err
}

func TestABatchCountedLogsNothing(t *testing.T) {
	logs, out, err := run(t, t.Context(), stubExecutor{out: count_takes_usecase.Out{From: 4, Takes: 2}})

	require.NoError(t, err)
	assert.Equal(t, count_takes_usecase.Out{From: 4, Takes: 2}, out)
	assert.Empty(t, logs)
}

func TestTheBeginningIsLoggedWithWhereItBegan(t *testing.T) {
	logs, _, err := run(t, t.Context(), stubExecutor{out: count_takes_usecase.Out{Began: true, From: 42}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=INFO msg="the standings count the takes from planet's log" from=42`)
}

func TestAFailureIsLoggedUnlessTheProcessIsStopping(t *testing.T) {
	cause := errors.New("planet is down")

	logs, _, err := run(t, t.Context(), stubExecutor{out: count_takes_usecase.Out{From: 7}, err: cause})
	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=ERROR msg="failed to count the takes into the standings" from=7 error="planet is down"`)

	stopping, stop := context.WithCancel(t.Context())
	stop()
	logs, _, err = run(t, stopping, stubExecutor{err: cause})
	require.ErrorIs(t, err, cause)
	assert.Empty(t, logs)
}
