package audit_warn_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/warn_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/warn_player_usecase/audit_warn"
)

const account = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

type stubUseCase struct {
	out warn_player_usecase.Out
	err error
}

func (s stubUseCase) Execute(context.Context, warn_player_usecase.In) (warn_player_usecase.Out, error) {
	return s.out, s.err
}

func TestAWarningIsLoggedWithTheAccountItLandedOn(t *testing.T) {
	var logs bytes.Buffer
	want := warn_player_usecase.Out{Account: account, WarnedAt: time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)}
	useCase := audit_warn.New(stubUseCase{out: want}, slog.New(slog.NewTextHandler(&logs, nil)))

	out, err := useCase.Execute(t.Context(), warn_player_usecase.In{Account: "0B7E5B6C-8F3A-4D2E-9C1A-2F6D8E4B7A10"})
	require.NoError(t, err)

	assert.Equal(t, want, out)
	assert.Contains(t, logs.String(), `level=WARN msg="admin warning" askedAccount=0B7E5B6C-8F3A-4D2E-9C1A-2F6D8E4B7A10 account=`+account+
		` warnedAt=2026-11-12T09:00:00.000Z alreadyWarned=false`)
}

func TestARefusedWarningIsLoggedToo(t *testing.T) {
	var logs bytes.Buffer
	useCase := audit_warn.New(stubUseCase{out: warn_player_usecase.Out{Account: account}, err: warn_player_usecase.ErrNotEnforced},
		slog.New(slog.NewTextHandler(&logs, nil)))

	_, err := useCase.Execute(t.Context(), warn_player_usecase.In{Account: account})

	require.ErrorIs(t, err, warn_player_usecase.ErrNotEnforced)
	assert.Contains(t, logs.String(), `level=WARN msg="admin warning failed" askedAccount=`+account)
	assert.Contains(t, logs.String(), "bans.enforce is off")
}
