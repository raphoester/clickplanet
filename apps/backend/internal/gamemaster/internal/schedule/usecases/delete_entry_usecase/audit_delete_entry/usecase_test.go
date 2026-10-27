package audit_delete_entry_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/delete_entry_usecase/audit_delete_entry"
)

type stubUseCase struct{ err error }

func (s stubUseCase) Execute(context.Context, schedule.EntryID) error { return s.err }

var first = schedule.EntryID(uuid.UUID{15: 1})

func TestEveryDeleteIsLogged(t *testing.T) {
	var logs bytes.Buffer

	require.NoError(t, audit_delete_entry.New(stubUseCase{}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), first))

	assert.Contains(t, logs.String(), `level=WARN msg="admin schedule entry deleted" entry=00000000-0000-0000-0000-000000000001`)
}

func TestARefusedDeleteIsLoggedWithWhyAndPassedOn(t *testing.T) {
	var logs bytes.Buffer

	err := audit_delete_entry.New(stubUseCase{err: schedule.ErrStarted}, slog.New(slog.NewTextHandler(&logs, nil))).
		Execute(t.Context(), first)

	require.ErrorIs(t, err, schedule.ErrStarted)
	assert.Contains(t, logs.String(), `level=WARN msg="admin schedule delete refused"`)
}
