package log_play_rules_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/play_rules_usecase/log_play_rules"
)

type stubExecutor struct {
	taken rules.Snapshot
	took  bool
	err   error
}

func (s stubExecutor) Execute(context.Context) (rules.Snapshot, bool, error) {
	return s.taken, s.took, s.err
}

func (s stubExecutor) Play(context.Context, rules.Choice) (rules.Snapshot, bool, error) {
	return s.taken, s.took, s.err
}

func TestASetTakenIsLoggedWithItsRules(t *testing.T) {
	var logs bytes.Buffer
	logged := log_play_rules.New(stubExecutor{taken: rules.Seeded(), took: true}, slog.New(slog.NewTextHandler(&logs, nil)))

	_, took, err := logged.Execute(t.Context())

	require.NoError(t, err)
	assert.True(t, took)
	assert.Contains(t, logs.String(), `level=WARN msg="antibot rules in force" set=default rules=`)
	assert.Contains(t, logs.String(), `minSuspects`)
}

func TestASetAlreadyInForceSaysNothing(t *testing.T) {
	var logs bytes.Buffer
	logged := log_play_rules.New(stubExecutor{}, slog.New(slog.NewTextHandler(&logs, nil)))

	_, _, err := logged.Execute(t.Context())

	require.NoError(t, err)
	assert.Empty(t, logs.String())
}

func TestAFailureIsLoggedUnlessTheProcessStops(t *testing.T) {
	var logs bytes.Buffer
	down := errors.New("postgres is down")
	logged := log_play_rules.New(stubExecutor{err: down}, slog.New(slog.NewTextHandler(&logs, nil)))

	_, _, err := logged.Execute(t.Context())
	require.ErrorIs(t, err, down)
	assert.Contains(t, logs.String(), `level=ERROR msg="failed to play the antibot rules" error="postgres is down"`)

	logs.Reset()
	stopped, cancel := context.WithCancel(t.Context())
	cancel()
	_, _, err = logged.Execute(stopped)
	require.ErrorIs(t, err, down)
	assert.Empty(t, logs.String())
}

func TestAChoiceRefusedIsLogged(t *testing.T) {
	var logs bytes.Buffer
	logged := log_play_rules.New(stubExecutor{taken: rules.Seeded(), err: rules.ErrUnknownSet},
		slog.New(slog.NewTextHandler(&logs, nil)))

	_, _, err := logged.Play(t.Context(), rules.Choice{Set: "rush"})

	require.ErrorIs(t, err, rules.ErrUnknownSet)
	assert.Contains(t, logs.String(), `level=ERROR msg="failed to play the antibot rules" error="no rule set has this name"`)
}
