package log_play_rules_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules/usecases/play_rules_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules/usecases/play_rules_usecase/log_play_rules"
)

type stubExecutor struct {
	played play_rules_usecase.Played
	took   bool
	err    error
}

func (s stubExecutor) Execute(context.Context) (play_rules_usecase.Played, bool, error) {
	return s.played, s.took, s.err
}

func (s stubExecutor) Play(context.Context, rules.Choice) (play_rules_usecase.Played, bool, error) {
	return s.played, s.took, s.err
}

func run(ctx context.Context, inner stubExecutor) (string, error) {
	var logs bytes.Buffer
	_, _, err := log_play_rules.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(ctx)
	return logs.String(), err
}

var battle = rules.Window{
	StartsAt: time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC),
}

func TestAPlayThatTookNothingLogsNothing(t *testing.T) {
	logs, err := run(t.Context(), stubExecutor{played: play_rules_usecase.Played{Set: rules.Seeded()}})

	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestTheSetTakenIsLoggedWithItsRulesAndItsWindow(t *testing.T) {
	logs, err := run(t.Context(), stubExecutor{
		played: play_rules_usecase.Played{Set: rules.Battle("rush", 3), Window: battle}, took: true,
	})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="season rules in force" set=rush rules=`)
	assert.Contains(t, logs, `\"times\":3`)
	assert.Contains(t, logs, `startsAt=2026-10-31T21:00:00.000Z endsAt=2026-10-31T23:00:00.000Z`)
}

func TestAFailureIsLoggedButNotOneTheShutdownCut(t *testing.T) {
	cause := errors.New("postgres is down")

	logs, err := run(t.Context(), stubExecutor{err: cause})
	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=ERROR msg="failed to play the season rules" error="postgres is down"`)

	cut, cancel := context.WithCancel(t.Context())
	cancel()
	logs, err = run(cut, stubExecutor{err: cause})
	require.ErrorIs(t, err, cause)
	assert.Empty(t, logs)
}

func TestAChoiceRefusedIsLogged(t *testing.T) {
	var logs bytes.Buffer
	_, _, err := log_play_rules.New(stubExecutor{played: play_rules_usecase.Played{Set: rules.Seeded()}, err: rules.ErrUnknownSet},
		slog.New(slog.NewTextHandler(&logs, nil))).Play(t.Context(), rules.Choice{Set: "rush"})

	require.ErrorIs(t, err, rules.ErrUnknownSet)
	assert.Contains(t, logs.String(), `level=ERROR msg="failed to play the season rules" error="no rule set has this name"`)
}
