package log_enforce_rules_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/enforce_rules_usecase/log_enforce_rules"
)

type stubExecutor struct {
	enforced rules.Snapshot
	err      error
}

func (s stubExecutor) Execute(context.Context) (rules.Snapshot, error) { return s.enforced, s.err }

func run(ctx context.Context, inner stubExecutor) (string, error) {
	var logs bytes.Buffer
	_, err := log_enforce_rules.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(ctx)
	return logs.String(), err
}

func TestAPassThatChangedNothingLogsNothing(t *testing.T) {
	logs, err := run(t.Context(), stubExecutor{})

	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestTheRulesEnforcedAreLoggedWithTheirValues(t *testing.T) {
	rush := rules.TestSnapshot("rush")
	rush.Tempo = 3
	rush.Bonuses.Boxes = bonuses.Pace{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30}

	logs, err := run(t.Context(), stubExecutor{enforced: rush})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="rules in force" set=rush rules=`)
	assert.Contains(t, logs, `\"tempo\":3`)
	assert.Contains(t, logs, `\"minInterval\":\"60s\"`)
	assert.NotContains(t, logs, `\"name\"`, "the set is named once")
}

func TestAFailureIsLoggedButNotOneTheShutdownCut(t *testing.T) {
	cause := errors.New("postgres is down")

	logs, err := run(t.Context(), stubExecutor{err: cause})
	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=ERROR msg="failed to enforce the rules" error="postgres is down"`)

	cut, cancel := context.WithCancel(t.Context())
	cancel()
	logs, err = run(cut, stubExecutor{err: cause})
	require.ErrorIs(t, err, cause)
	assert.Empty(t, logs)
}
