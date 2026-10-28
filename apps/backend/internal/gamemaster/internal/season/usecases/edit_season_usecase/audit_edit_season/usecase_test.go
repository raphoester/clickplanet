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

var moved = season.Draft{Length: 35 * 24 * time.Hour}

func TestEveryEditIsLoggedWithTheNewWindow(t *testing.T) {
	var logs bytes.Buffer

	require.NoError(t, audit_edit_season.New(stubUseCase{}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), 0, moved))

	assert.Contains(t, logs.String(), `level=WARN msg="admin season edited" season=0 break=0s length=840h0m0s`)
}

func TestARefusalIsLoggedWithWhyAndPassedOn(t *testing.T) {
	var logs bytes.Buffer

	err := audit_edit_season.New(stubUseCase{err: season.ErrEnded}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), 0, moved)

	require.ErrorIs(t, err, season.ErrEnded)
	assert.Contains(t, logs.String(), `level=WARN msg="admin season edit refused" season=0`)
}
