package audit_edit_season_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/edit_season_usecase/audit_edit_season"
)

type stubUseCase struct{ err error }

func (s stubUseCase) Execute(context.Context, season.Number, season.Draft) error { return s.err }

var moved = season.Draft{StartsAt: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, 11, 7, 23, 0, 0, 0, time.UTC)}

func TestEveryEditIsLoggedWithTheNewWindow(t *testing.T) {
	var logs bytes.Buffer

	require.NoError(t, audit_edit_season.New(stubUseCase{}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), 0, moved))

	assert.Contains(t, logs.String(), `level=WARN msg="admin season edited" season=0 starts_at=2026-10-03T00:00:00Z ends_at=2026-11-07T23:00:00Z`)
}

func TestARefusalIsLoggedWithWhyAndPassedOn(t *testing.T) {
	var logs bytes.Buffer

	err := audit_edit_season.New(stubUseCase{err: season.ErrEnded}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), 0, moved)

	require.ErrorIs(t, err, season.ErrEnded)
	assert.Contains(t, logs.String(), `level=WARN msg="admin season edit refused" season=0`)
}
