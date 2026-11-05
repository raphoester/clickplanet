package log_command_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/commands/log_command"
)

type stubExecutor struct {
	applied bool
	err     error
}

func (s stubExecutor) Execute(context.Context, string) (bool, error) { return s.applied, s.err }

func TestACommandAppliedIsLoggedAndOneThatChangedNothingIsNot(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))

	applied, err := log_command.New[string]("open_book", stubExecutor{applied: true}, logger).Execute(t.Context(), "season-0")
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, logs.String(), "scoring command applied")
	assert.Contains(t, logs.String(), "open_book")

	logs.Reset()
	_, err = log_command.New[string]("open_book", stubExecutor{}, logger).Execute(t.Context(), "season-0")
	require.NoError(t, err)
	assert.Empty(t, logs.String())
}
