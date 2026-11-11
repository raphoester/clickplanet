package cprules_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

type stubSaving struct{ err error }

func (s stubSaving) Execute(context.Context, snapshot) error { return s.err }

func audited(inner stubSaving, logs *bytes.Buffer) *cprules.AuditedSave[*planetv1.Rules] {
	return cprules.NewAuditedSave(inner, slog.New(slog.NewTextHandler(logs, nil)),
		cprules.SaveLog{Saved: "admin offer rule set saved", Refused: "admin offer rule set refused"})
}

func TestEverySaveIsLoggedWithItsValues(t *testing.T) {
	var logs bytes.Buffer

	require.NoError(t, audited(stubSaving{}, &logs).Execute(t.Context(), rush()))

	assert.Contains(t, logs.String(), `level=WARN msg="admin offer rule set saved" set=rush rules=`)
	assert.Contains(t, logs.String(), `\"tempo\":3`)
}

func TestARefusedSaveIsLoggedWithWhyAndPassedOn(t *testing.T) {
	var logs bytes.Buffer
	refused := fmt.Errorf("cannot keep it: %w", cprules.ErrInvalidSet)

	err := audited(stubSaving{err: refused}, &logs).Execute(t.Context(), rush())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, logs.String(), `level=WARN msg="admin offer rule set refused" set=rush rules=`)
	assert.Contains(t, logs.String(), `error="cannot keep it: this rule set cannot play"`)
}
