package audit_mute_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/mute_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/mute_usecase/audit_mute"
)

type stubExecutor struct {
	mute mutes.Mute
	err  error
}

func (s stubExecutor) Execute(context.Context, mute_usecase.In) (mutes.Mute, error) {
	return s.mute, s.err
}

var bully = messages.AccountID{15: 1}

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := audit_mute.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).
		Execute(t.Context(), mute_usecase.In{Account: bully, Duration: time.Hour})
	return logs.String(), err
}

func TestEveryMuteIsLoggedWithTheNetworkItHolds(t *testing.T) {
	at := time.Date(2026, 10, 7, 20, 0, 0, 0, time.UTC)
	mute := mutes.NewMute(mutes.MuteID{1}, mutes.CallerOf(bully, "2a01:e0a:1:2::/64"), at, time.Hour)

	logs, err := run(t, stubExecutor{mute: mute})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin chat mute" account=00000000-0000-0000-0000-000000000001 duration=1h0m0s scope=2a01:e0a:1:2::/64 mutedUntil=2026-10-07T21:00:00.000Z`)
}

func TestARefusedMuteIsLoggedToo(t *testing.T) {
	cause := errors.New("postgres is down")

	logs, err := run(t, stubExecutor{err: cause})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=WARN msg="admin chat mute failed" account=00000000-0000-0000-0000-000000000001`)
}
