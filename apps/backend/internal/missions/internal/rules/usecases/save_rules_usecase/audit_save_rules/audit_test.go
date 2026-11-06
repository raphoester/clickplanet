package audit_save_rules_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/rules/usecases/save_rules_usecase/audit_save_rules"
)

type stubUseCase struct{ err error }

func (s stubUseCase) Execute(context.Context, *missionsv1.Rules) error { return s.err }

var in = &missionsv1.Rules{FlagMissions: &missionsv1.FlagMissionRules{GoalPercent: 40, LeastGoal: 300, FewestPoints: 2, MostPoints: 8}}

func TestASaveIsLoggedWithTheNumbersItKept(t *testing.T) {
	var logs bytes.Buffer

	require.NoError(t, audit_save_rules.New(stubUseCase{}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), in))

	assert.Contains(t, logs.String(), `level=WARN msg="admin missions rules saved" rules=`)
	assert.Contains(t, logs.String(), `goalPercent`)
}

func TestARefusedSaveIsLoggedToo(t *testing.T) {
	var logs bytes.Buffer
	refused := fmt.Errorf("%w: flagMissions: no", rules.ErrInvalidRules)

	err := audit_save_rules.New(stubUseCase{err: refused}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), in)

	require.ErrorIs(t, err, rules.ErrInvalidRules)
	assert.Contains(t, logs.String(), `level=WARN msg="admin missions rules refused" rules=`)
	assert.Contains(t, logs.String(), `error="these rules cannot play: flagMissions: no"`)
}
