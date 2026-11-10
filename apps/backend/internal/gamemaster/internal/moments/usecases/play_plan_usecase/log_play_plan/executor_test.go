package log_play_plan_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_plan_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_plan_usecase/log_play_plan"
)

type stubExecutor struct {
	err error
}

func (s stubExecutor) Execute(context.Context) error {
	return s.err
}

func run(ctx context.Context, inner stubExecutor) (string, error) {
	var logs bytes.Buffer
	err := log_play_plan.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(ctx)
	return logs.String(), err
}

func TestAPlanThatCannotBeReadIsLogged(t *testing.T) {
	down := fmt.Errorf("%w: %w", play_plan_usecase.ErrUnplanned, errors.New("postgres is down"))

	logs, err := run(t.Context(), stubExecutor{err: down})

	require.ErrorIs(t, err, down)
	assert.Contains(t, logs, `level=ERROR msg="failed to plan the books" error="could not plan the books: postgres is down"`)
}

func TestAStepThatFailedIsLeftToItsOwnLog(t *testing.T) {
	down := errors.New("redis is down")

	logs, err := run(t.Context(), stubExecutor{err: down})

	require.ErrorIs(t, err, down)
	assert.Empty(t, logs)
}

func TestAPlanTheShutdownCutIsNotLogged(t *testing.T) {
	cut, cancel := context.WithCancel(t.Context())
	cancel()

	logs, err := run(cut, stubExecutor{err: fmt.Errorf("%w: %w", play_plan_usecase.ErrUnplanned, context.Canceled)})

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, logs)
}
