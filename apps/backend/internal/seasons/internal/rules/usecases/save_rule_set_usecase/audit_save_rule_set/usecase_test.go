package audit_save_rule_set_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rules/usecases/save_rule_set_usecase/audit_save_rule_set"
)

type stubUseCase struct{ err error }

func (s stubUseCase) Execute(context.Context, rules.Snapshot) error { return s.err }

func TestEverySaveIsLoggedWithItsValues(t *testing.T) {
	var logs bytes.Buffer

	err := audit_save_rule_set.New(stubUseCase{}, slog.New(slog.NewTextHandler(&logs, nil))).
		Execute(t.Context(), rules.Battle("rush", 3))

	require.NoError(t, err)
	assert.Contains(t, logs.String(), `level=WARN msg="admin season rule set saved" set=rush rules=`)
	assert.Contains(t, logs.String(), `\"times\":3`)
}

func TestARefusedSaveIsLoggedWithWhyAndPassedOn(t *testing.T) {
	var logs bytes.Buffer
	cause := errors.New("no times")

	err := audit_save_rule_set.New(stubUseCase{err: cause}, slog.New(slog.NewTextHandler(&logs, nil))).
		Execute(t.Context(), rules.Battle("rush", 3))

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs.String(), `level=WARN msg="admin season rule set refused" set=rush`)
	assert.Contains(t, logs.String(), `error="no times"`)
}
