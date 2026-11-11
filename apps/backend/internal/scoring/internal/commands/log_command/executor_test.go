package log_command_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/commands/log_command"
)

type stubExecutor struct {
	answer bool
	err    error
}

func (s stubExecutor) Execute(context.Context, string) (bool, error) { return s.answer, s.err }

func execute(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()
	var logs bytes.Buffer
	_, err := log_command.New[string, bool]("open_book", inner, slog.New(slog.NewTextHandler(&logs, nil))).
		Execute(t.Context(), "season-0")
	return logs.String(), err
}

func TestEachAnswerIsLoggedAtInfo(t *testing.T) {
	for _, answer := range []bool{true, false} {
		logs, err := execute(t, stubExecutor{answer: answer})

		require.NoError(t, err)
		assert.Contains(t, logs, "level=INFO")
		assert.Contains(t, logs, "open_book")
		assert.Contains(t, logs, fmt.Sprintf("answer=%t", answer))
	}
}

func TestARefusalIsLoggedAtWarnAndAFailureIsLeftToTheErrorNet(t *testing.T) {
	refusal := fmt.Errorf("%w: book season-0: %w", books.ErrRefused, books.ErrMissed)
	logs, err := execute(t, stubExecutor{err: refusal})
	require.ErrorIs(t, err, refusal)
	assert.Contains(t, logs, "level=WARN")

	failure := errors.New("postgres is down")
	logs, err = execute(t, stubExecutor{err: failure})
	require.ErrorIs(t, err, failure)
	assert.Empty(t, logs)
}
