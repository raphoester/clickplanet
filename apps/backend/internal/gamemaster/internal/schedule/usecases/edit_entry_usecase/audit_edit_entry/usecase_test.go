package audit_edit_entry_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/edit_entry_usecase/audit_edit_entry"
)

type stubUseCase struct{ err error }

func (s stubUseCase) Execute(context.Context, schedule.EntryID, schedule.Draft) error { return s.err }

var (
	finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	first  = schedule.EntryID(uuid.UUID{15: 1})
	calm   = schedule.Draft{
		Set: "calm", Kind: schedule.Plain, Description: "A quieter antibot.", StartsAt: finale, EndsAt: finale.Add(time.Hour),
		Priority: 2,
	}
)

func TestEveryEditIsLoggedWithItsValues(t *testing.T) {
	var logs bytes.Buffer

	err := audit_edit_entry.New(stubUseCase{}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), first, calm)

	require.NoError(t, err)
	assert.Contains(t, logs.String(), `level=WARN msg="admin schedule entry edited" set=calm kind=plain description="A quieter antibot."`)
	assert.Contains(t, logs.String(), `entry=00000000-0000-0000-0000-000000000001`)
}

func TestARefusedEditIsLoggedWithWhyAndPassedOn(t *testing.T) {
	var logs bytes.Buffer

	err := audit_edit_entry.New(stubUseCase{err: schedule.ErrStarted}, slog.New(slog.NewTextHandler(&logs, nil))).
		Execute(t.Context(), first, calm)

	require.ErrorIs(t, err, schedule.ErrStarted)
	assert.Contains(t, logs.String(), `level=WARN msg="admin schedule edit refused" set=calm`)
}
