package audit_save_rule_set_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/save_rule_set_usecase/audit_save_rule_set"
)

type stubUseCase struct{ err error }

func (s stubUseCase) Execute(context.Context, rules.Snapshot) error { return s.err }

var rush = rules.Named("rush", func(r *antibotv1.Rules) { r.Cohort.CertainMembers = 50 })

func TestASaveIsLoggedWithTheSetItKept(t *testing.T) {
	var logs bytes.Buffer
	audited := audit_save_rule_set.New(stubUseCase{}, slog.New(slog.NewTextHandler(&logs, nil)))

	require.NoError(t, audited.Execute(t.Context(), rush))

	assert.Contains(t, logs.String(), `level=WARN msg="admin antibot rule set saved" set=rush rules=`)
	assert.Contains(t, logs.String(), `certainMembers`)
}

func TestARefusedSaveIsLoggedToo(t *testing.T) {
	var logs bytes.Buffer
	refused := fmt.Errorf("cannot keep it: %w", rules.ErrInvalidSet)
	audited := audit_save_rule_set.New(stubUseCase{err: refused}, slog.New(slog.NewTextHandler(&logs, nil)))

	err := audited.Execute(t.Context(), rush)

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, logs.String(), `level=WARN msg="admin antibot rule set refused" set=rush rules=`)
	assert.Contains(t, logs.String(), `error="cannot keep it: this rule set cannot play"`)
}
