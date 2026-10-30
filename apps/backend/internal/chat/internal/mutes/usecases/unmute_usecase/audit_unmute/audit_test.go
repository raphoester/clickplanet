package audit_unmute_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/unmute_usecase/audit_unmute"
)

type stubExecutor struct {
	until time.Time
	err   error
}

func (s stubExecutor) Execute(context.Context, messages.AccountID) (time.Time, error) {
	return s.until, s.err
}

var bully = messages.AccountID{15: 1}

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := audit_unmute.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), bully)
	return logs.String(), err
}

func TestEveryUnmuteIsLoggedWithWhenTheMuteWouldHaveEnded(t *testing.T) {
	logs, err := run(t, stubExecutor{until: time.Date(2026, 10, 30, 21, 0, 0, 0, time.UTC)})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin chat unmute" account=00000000-0000-0000-0000-000000000001 mutedUntil=2026-10-30T21:00:00.000Z`)
}

func TestARefusedUnmuteIsLoggedToo(t *testing.T) {
	cause := errors.New("postgres is down")

	logs, err := run(t, stubExecutor{err: cause})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=WARN msg="admin chat unmute failed" account=00000000-0000-0000-0000-000000000001`)
}
