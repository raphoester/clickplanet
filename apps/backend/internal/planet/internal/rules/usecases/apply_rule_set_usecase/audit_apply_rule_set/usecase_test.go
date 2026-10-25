package audit_apply_rule_set_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/apply_rule_set_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/apply_rule_set_usecase/audit_apply_rule_set"
)

type stubUseCase struct{ err error }

func (s stubUseCase) Execute(context.Context, apply_rule_set_usecase.In) error { return s.err }

var asked = apply_rule_set_usecase.In{Request: "season-0:finale", Set: "rush"}

func TestEveryRequestIsLoggedWithItsSet(t *testing.T) {
	var logs bytes.Buffer

	err := audit_apply_rule_set.New(stubUseCase{}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), asked)

	require.NoError(t, err)
	assert.Contains(t, logs.String(), `level=WARN msg="rule set applied" request=season-0:finale set=rush`)
}

func TestARefusedRequestIsLoggedWithWhyAndPassedOn(t *testing.T) {
	var logs bytes.Buffer
	cause := errors.New("no such set")

	err := audit_apply_rule_set.New(stubUseCase{err: cause}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), asked)

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs.String(), `level=WARN msg="rule set refused" request=season-0:finale set=rush error="no such set"`)
}
