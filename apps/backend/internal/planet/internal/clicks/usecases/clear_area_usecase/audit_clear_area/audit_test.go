package audit_clear_area_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/clear_area_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/clear_area_usecase/audit_clear_area"
)

type stubUseCase struct {
	out clear_area_usecase.Out
	err error
}

func (s stubUseCase) Execute(context.Context, clear_area_usecase.In) (clear_area_usecase.Out, error) {
	return s.out, s.err
}

func run(t *testing.T, inner stubUseCase) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := audit_clear_area.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).
		Execute(t.Context(), clear_area_usecase.In{Tile: 42, Rings: 3})
	return logs.String(), err
}

func TestEveryClearIsLogged(t *testing.T) {
	logs, err := run(t, stubUseCase{out: clear_area_usecase.Out{Tiles: 37, Owned: 20, Cleared: 20}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin area clear" tile=42 rings=3 dryRun=false tiles=37 owned=20 cleared=20`)
}

func TestAFailedClearIsLoggedToo(t *testing.T) {
	cause := errors.New("the map is closed")

	logs, err := run(t, stubUseCase{err: cause})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=WARN msg="admin area clear failed" tile=42`)
}
