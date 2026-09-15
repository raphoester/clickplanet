package audit_ban_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/ban_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/ban_player_usecase/audit_ban"
)

type stubUseCase struct {
	out ban_player_usecase.Out
	err error
}

func (s stubUseCase) Execute(context.Context, ban_player_usecase.In) (ban_player_usecase.Out, error) {
	return s.out, s.err
}

func TestABanIsLoggedWithTheScopeItLandedOn(t *testing.T) {
	var logs bytes.Buffer
	want := ban_player_usecase.Out{Scope: "2001:db8::/64", Offence: 2, Enforced: true}
	useCase := audit_ban.New(stubUseCase{out: want}, slog.New(slog.NewTextHandler(&logs, nil)))

	out, err := useCase.Execute(t.Context(), ban_player_usecase.In{Scope: "2001:db8::1"})
	require.NoError(t, err)

	assert.Equal(t, want, out)
	assert.Contains(t, logs.String(), `level=WARN msg="admin ban" asked=2001:db8::1 duration=0s scope=2001:db8::/64 offence=2`)
	assert.Contains(t, logs.String(), "enforced=true")
}

func TestARefusedBanIsLoggedToo(t *testing.T) {
	var logs bytes.Buffer
	cause := errors.New("not an address or a scope")
	useCase := audit_ban.New(stubUseCase{err: cause}, slog.New(slog.NewTextHandler(&logs, nil)))

	_, err := useCase.Execute(t.Context(), ban_player_usecase.In{Scope: "bot"})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs.String(), `msg="admin ban failed" asked=bot`)
}
