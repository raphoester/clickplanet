package audit_remove_messages_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/usecases/remove_messages_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/usecases/remove_messages_usecase/audit_remove_messages"
)

type stubUseCase struct {
	removedAt time.Time
	err       error
}

func (s stubUseCase) Execute(context.Context, remove_messages_usecase.In) (time.Time, error) {
	return s.removedAt, s.err
}

func run(t *testing.T, inner stubUseCase) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := audit_remove_messages.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(),
		remove_messages_usecase.In{Removals: []remove_messages_usecase.Removal{
			{Message: "insult", Reason: "a slur"},
			{Message: "spam", Reason: "spam, no offence"},
		}})
	return logs.String(), err
}

func TestEveryMessageRemovedIsLoggedAsARemovalOfOne(t *testing.T) {
	logs, err := run(t, stubUseCase{removedAt: time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin chat message removal" message=insult reason="a slur" batch=2`)
	assert.Contains(t, logs, `level=WARN msg="admin chat message removal" message=spam reason="spam, no offence" batch=2`)
}

func TestARefusedBatchIsLoggedOnceWithEveryMessage(t *testing.T) {
	cause := errors.New("postgres is down")

	logs, err := run(t, stubUseCase{err: cause})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=WARN msg="admin chat message removals failed" messages="[insult spam]"`)
	assert.NotContains(t, logs, `msg="admin chat message removal" `)
}
