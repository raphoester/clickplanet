package audit_delete_season_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/delete_season_usecase/audit_delete_season"
)

type stubUseCase struct{ err error }

func (s stubUseCase) Execute(context.Context, season.Number) error { return s.err }

func TestEveryDeleteIsLogged(t *testing.T) {
	var logs bytes.Buffer

	require.NoError(t, audit_delete_season.New(stubUseCase{}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), 2))

	assert.Contains(t, logs.String(), `level=WARN msg="admin season deleted" season=2`)
}

func TestARefusalIsLoggedWithWhyAndPassedOn(t *testing.T) {
	var logs bytes.Buffer

	err := audit_delete_season.New(stubUseCase{err: season.ErrStarted}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), 0)

	require.ErrorIs(t, err, season.ErrStarted)
	assert.Contains(t, logs.String(), `level=WARN msg="admin season delete refused" season=0`)
}
