package mark_seen_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/mark_seen_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen/usecases/mark_seen_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

var ada = messages.AccountID{15: 1}

type stubUseCase struct {
	err   error
	asked []mark_seen_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in mark_seen_usecase.In) error {
	s.asked = append(s.asked, in)
	return s.err
}

func markSeen(t *testing.T, useCase *stubUseCase, unixMs int64) error {
	t.Helper()

	ctx := cpctx.AddAccountToContext(t.Context(), ada.String())
	_, err := mark_seen_handler.New(useCase).MarkSeen(ctx, connect.NewRequest(&chatv1.MarkSeenRequest{SeenUntilUnixMs: unixMs}))
	if err != nil {
		return fmt.Errorf("MarkSeen failed: %w", err)
	}
	return nil
}

func TestTheCallersAccountAndTheTimeAreHandedOn(t *testing.T) {
	useCase := &stubUseCase{}
	at := time.Date(2026, 10, 4, 12, 0, 0, 123_000_000, time.UTC)

	require.NoError(t, markSeen(t, useCase, at.UnixMilli()))

	assert.Equal(t, []mark_seen_usecase.In{{Account: ada, At: at}}, useCase.asked)
}

func TestTheRefusalsHaveTheirCodes(t *testing.T) {
	for want, refusal := range map[connect.Code]error{
		connect.CodeUnauthenticated: messages.ErrNoAccount,
		connect.CodeInvalidArgument: fmt.Errorf("wrapped: %w", seen.ErrNoTime),
	} {
		assert.Equal(t, want, connect.CodeOf(markSeen(t, &stubUseCase{err: refusal}, 1)), "%v", refusal)
	}
}

func TestAStoreFailureIsLeftToTheErrorNet(t *testing.T) {
	err := markSeen(t, &stubUseCase{err: errors.New("postgres is down")}, 1)

	assert.ErrorContains(t, err, "postgres is down")
}
