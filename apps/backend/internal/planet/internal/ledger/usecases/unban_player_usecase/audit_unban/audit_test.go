package audit_unban_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/unban_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/unban_player_usecase/audit_unban"
)

type stubUseCase struct {
	out unban_player_usecase.Out
	err error
}

func (s stubUseCase) Execute(context.Context, unban_player_usecase.In) (unban_player_usecase.Out, error) {
	return s.out, s.err
}

func TestAnUnbanIsLoggedWithTheBanItLifted(t *testing.T) {
	var logs bytes.Buffer
	want := unban_player_usecase.Out{Scope: "2001:db8::/64", Offence: 1, Until: time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)}
	useCase := audit_unban.New(stubUseCase{out: want}, slog.New(slog.NewTextHandler(&logs, nil)))

	out, err := useCase.Execute(t.Context(), unban_player_usecase.In{Scope: "2001:db8::1"})
	require.NoError(t, err)

	assert.Equal(t, want, out)
	assert.Contains(t, logs.String(),
		`level=WARN msg="admin unban" asked=2001:db8::1 askedAccount="" scope=2001:db8::/64 account="" offence=1 bannedUntil=2026-10-09T12:00:00.000Z`)
}

func TestARefusedUnbanIsLoggedToo(t *testing.T) {
	var logs bytes.Buffer
	useCase := audit_unban.New(stubUseCase{err: unban_player_usecase.ErrNotBanned}, slog.New(slog.NewTextHandler(&logs, nil)))

	_, err := useCase.Execute(t.Context(), unban_player_usecase.In{Scope: "1.2.3.4"})

	require.ErrorIs(t, err, unban_player_usecase.ErrNotBanned)
	assert.Contains(t, logs.String(), `level=WARN msg="admin unban failed" asked=1.2.3.4`)
}
