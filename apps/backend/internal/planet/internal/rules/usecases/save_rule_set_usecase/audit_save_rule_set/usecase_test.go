package audit_save_rule_set_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/save_rule_set_usecase/audit_save_rule_set"
)

type stubUseCase struct{ err error }

func (s stubUseCase) Execute(context.Context, rules.Snapshot) error { return s.err }

var saved = rules.Snapshot{
	Name:  "rush",
	Tempo: 3,
	Boxes: bonuses.Pace{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30},
}

func TestEverySaveIsLoggedWithItsValues(t *testing.T) {
	var logs bytes.Buffer

	err := audit_save_rule_set.New(stubUseCase{}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), saved)

	require.NoError(t, err)
	assert.Contains(t, logs.String(), `level=WARN msg="admin rule set saved" set=rush tempo=3 boxMinInterval=1m0s boxMaxInterval=2m0s boxMaxChargesPerHour=30`)
}

func TestARefusedSaveIsLoggedWithWhyAndPassedOn(t *testing.T) {
	var logs bytes.Buffer
	cause := errors.New("no tempo")

	err := audit_save_rule_set.New(stubUseCase{err: cause}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), saved)

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs.String(), `level=WARN msg="admin rule set refused" set=rush`)
	assert.Contains(t, logs.String(), `error="no tempo"`)
}
