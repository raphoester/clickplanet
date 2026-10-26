package log_play_rules_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/play_rules_usecase/log_play_rules"
)

type stubExecutor struct {
	taken rules.Snapshot
	took  bool
	err   error
}

func (s stubExecutor) Execute(context.Context) (rules.Snapshot, bool, error) {
	return s.taken, s.took, s.err
}

func run(ctx context.Context, inner stubExecutor) (string, error) {
	var logs bytes.Buffer
	_, _, err := log_play_rules.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(ctx)
	return logs.String(), err
}

func TestAPlayThatTookNothingLogsNothing(t *testing.T) {
	logs, err := run(t.Context(), stubExecutor{taken: rules.Seeded()})

	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestTheSetTakenIsLoggedWithItsRules(t *testing.T) {
	logs, err := run(t.Context(), stubExecutor{
		taken: rules.Named("rush", func(r *planetv1.Rules) { r.Clicks.Tempo = 3 }), took: true,
	})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="rules in force" set=rush rules=`)
	assert.Contains(t, logs, `\"tempo\":3`)
}

func TestAFailureIsLoggedButNotOneTheShutdownCut(t *testing.T) {
	cause := errors.New("postgres is down")

	logs, err := run(t.Context(), stubExecutor{err: cause})
	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=ERROR msg="failed to play the rules" error="postgres is down"`)

	cut, cancel := context.WithCancel(t.Context())
	cancel()
	logs, err = run(cut, stubExecutor{err: cause})
	require.ErrorIs(t, err, cause)
	assert.Empty(t, logs)
}
