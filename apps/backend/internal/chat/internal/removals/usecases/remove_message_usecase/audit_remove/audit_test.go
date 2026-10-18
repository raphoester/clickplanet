package audit_remove_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/usecases/remove_message_usecase/audit_remove"
)

type stubUseCase struct {
	removedAt time.Time
	err       error
}

func (s stubUseCase) Execute(context.Context, messages.MessageID) (time.Time, error) {
	return s.removedAt, s.err
}

func run(t *testing.T, inner stubUseCase) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := audit_remove.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), "insult")
	return logs.String(), err
}

func TestEveryRemovalIsLogged(t *testing.T) {
	logs, err := run(t, stubUseCase{removedAt: time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin chat message removal" message=insult`)
}

func TestARefusedRemovalIsLoggedToo(t *testing.T) {
	cause := errors.New("postgres is down")

	logs, err := run(t, stubUseCase{err: cause})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=WARN msg="admin chat message removal failed" message=insult`)
}
