package audit_add_season_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/add_season_usecase/audit_add_season"
)

type stubUseCase struct {
	number season.Number
	err    error
}

func (s stubUseCase) Execute(context.Context, season.Draft) (season.Number, error) {
	return s.number, s.err
}

var one = season.Draft{Break: time.Hour, Length: 28 * 24 * time.Hour}

func TestEverySeasonAddedIsLoggedWithItsWindow(t *testing.T) {
	var logs bytes.Buffer

	_, err := audit_add_season.New(stubUseCase{number: 1}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), one)

	require.NoError(t, err)
	assert.Contains(t, logs.String(), `level=WARN msg="admin season added" break=1h0m0s length=672h0m0s finale.length=0s finale.set="" finale.description="" finale.times=0 season=1`)
}

func TestARefusalIsLoggedWithWhyAndPassedOn(t *testing.T) {
	var logs bytes.Buffer

	_, err := audit_add_season.New(stubUseCase{err: season.ErrInvalidSeason}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), one)

	require.ErrorIs(t, err, season.ErrInvalidSeason)
	assert.Contains(t, logs.String(), `level=WARN msg="admin season refused" break=1h0m0s length=672h0m0s`)
}
