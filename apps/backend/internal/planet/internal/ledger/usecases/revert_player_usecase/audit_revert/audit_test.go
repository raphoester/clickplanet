package audit_revert_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/revert_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/revert_player_usecase/audit_revert"
)

type stubUseCase struct {
	out revert_player_usecase.Out
	err error
}

func (s stubUseCase) Execute(context.Context, revert_player_usecase.In) (revert_player_usecase.Out, error) {
	return s.out, s.err
}

func TestARevertIsLoggedWithItsCounts(t *testing.T) {
	var logs bytes.Buffer
	want := revert_player_usecase.Out{Scope: "9.9.9.9", Touched: 40, Held: 31, Restored: 31}
	useCase := audit_revert.New(stubUseCase{out: want}, slog.New(slog.NewTextHandler(&logs, nil)))

	out, err := useCase.Execute(t.Context(), revert_player_usecase.In{Scope: "9.9.9.9", DryRun: true})
	require.NoError(t, err)

	assert.Equal(t, want, out)
	assert.Contains(t, logs.String(),
		`level=WARN msg="admin player revert" asked=9.9.9.9 scope=9.9.9.9 dryRun=true touched=40 held=31 restored=31`)
}

func TestAFailureIsLoggedWithHowFarItGot(t *testing.T) {
	var logs bytes.Buffer
	cause := errors.New("revert interrupted")
	useCase := audit_revert.New(stubUseCase{out: revert_player_usecase.Out{Restored: 256}, err: cause},
		slog.New(slog.NewTextHandler(&logs, nil)))

	out, err := useCase.Execute(t.Context(), revert_player_usecase.In{Scope: "9.9.9.9"})

	require.ErrorIs(t, err, cause)
	assert.Equal(t, 256, out.Restored)
	assert.Contains(t, logs.String(), `msg="admin player revert failed"`)
	assert.Contains(t, logs.String(), `error="revert interrupted"`)
}
