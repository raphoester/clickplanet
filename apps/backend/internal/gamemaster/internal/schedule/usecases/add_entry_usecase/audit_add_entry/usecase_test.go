package audit_add_entry_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/add_entry_usecase/audit_add_entry"
)

type stubUseCase struct {
	id  schedule.EntryID
	err error
}

func (s stubUseCase) Execute(context.Context, schedule.Draft) (schedule.EntryID, error) {
	return s.id, s.err
}

var (
	finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	rush   = schedule.Draft{
		Set: "rush", Kind: schedule.Battle, Description: "Triple points.", StartsAt: finale, EndsAt: finale.Add(2 * time.Hour),
		Priority: 1, Times: 3,
	}
)

func TestEveryEntryAddedIsLoggedWithItsValues(t *testing.T) {
	var logs bytes.Buffer
	id := schedule.EntryID(uuid.UUID{15: 1})

	_, err := audit_add_entry.New(stubUseCase{id: id}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), rush)

	require.NoError(t, err)
	assert.Contains(t, logs.String(), `level=WARN msg="admin schedule entry added" set=rush kind=battle description="Triple points." `+
		`starts_at=2026-10-31T21:00:00Z ends_at=2026-10-31T23:00:00Z priority=1 times=3 entry=00000000-0000-0000-0000-000000000001`)
}

func TestARefusalIsLoggedWithWhyAndPassedOn(t *testing.T) {
	var logs bytes.Buffer

	_, err := audit_add_entry.New(stubUseCase{err: schedule.ErrUnknownSet}, slog.New(slog.NewTextHandler(&logs, nil))).
		Execute(t.Context(), rush)

	require.ErrorIs(t, err, schedule.ErrUnknownSet)
	assert.Contains(t, logs.String(), `level=WARN msg="admin schedule entry refused" set=rush`)
	assert.Contains(t, logs.String(), `error="keeps no rule set of this name"`)
}
