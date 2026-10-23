package audit_restore_area_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/restore_area_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/restore_area_usecase/audit_restore_area"
)

type stubUseCase struct {
	out restore_area_usecase.Out
	err error
}

func (s stubUseCase) Execute(context.Context, restore_area_usecase.In) (restore_area_usecase.Out, error) {
	return s.out, s.err
}

func run(t *testing.T, inner stubUseCase) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := audit_restore_area.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(),
		restore_area_usecase.In{Tile: 42, Rings: 3, Before: time.Date(2026, 10, 20, 11, 0, 0, 0, time.UTC), DryRun: true})
	return logs.String(), err
}

func TestEveryRestoreIsLoggedDryRunsToo(t *testing.T) {
	logs, err := run(t, stubUseCase{out: restore_area_usecase.Out{Tiles: 37, Changed: 12, PaintersTotal: 2}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin area restore" tile=42 rings=3 before=2026-10-20T11:00:00Z dryRun=true tiles=37 changed=12 restored=0 painters=2`)
}

func TestAFailedRestoreIsLoggedToo(t *testing.T) {
	cause := errors.New("the map is closed")

	logs, err := run(t, stubUseCase{err: cause})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=WARN msg="admin area restore failed" tile=42`)
}
