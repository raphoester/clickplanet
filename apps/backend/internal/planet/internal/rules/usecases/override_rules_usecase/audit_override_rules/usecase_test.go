package audit_override_rules_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/override_rules_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/override_rules_usecase/audit_override_rules"
)

type stubUseCase struct{ err error }

func (s stubUseCase) Execute(context.Context, override_rules_usecase.In) error { return s.err }

var asked = override_rules_usecase.In{Set: "rush", Until: time.Date(2026, 10, 31, 21, 30, 0, 0, time.UTC)}

func TestEveryOverrideIsLoggedWithItsSetAndEnd(t *testing.T) {
	var logs bytes.Buffer

	err := audit_override_rules.New(stubUseCase{}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), asked)

	require.NoError(t, err)
	assert.Contains(t, logs.String(), `level=WARN msg="admin rule override" set=rush until=2026-10-31T21:30:00Z`)
}

func TestARefusedOverrideIsLoggedWithWhyAndPassedOn(t *testing.T) {
	var logs bytes.Buffer
	cause := errors.New("no such set")

	err := audit_override_rules.New(stubUseCase{err: cause}, slog.New(slog.NewTextHandler(&logs, nil))).
		Execute(t.Context(), asked)

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs.String(),
		`level=WARN msg="admin rule override refused" set=rush until=2026-10-31T21:30:00Z error="no such set"`)
}
