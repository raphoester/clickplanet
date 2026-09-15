package get_history_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_history_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type stubUseCase []messages.Message

func (s stubUseCase) Execute(context.Context) []messages.Message { return s }

func getHistory(t *testing.T, useCase stubUseCase) *connect.Response[chatv1.GetHistoryResponse] {
	t.Helper()

	res, err := get_history_handler.New(useCase).
		GetHistory(t.Context(), connect.NewRequest(&chatv1.GetHistoryRequest{}))
	require.NoError(t, err)

	return res
}

func TestGetHistoryMapsEveryMessageInOrder(t *testing.T) {
	res := getHistory(t, stubUseCase{{ID: "message-1", Text: "hello"}, {ID: "message-2", Text: "planet"}})

	require.Len(t, res.Msg.GetMessages(), 2)
	assert.Equal(t, "message-1", res.Msg.GetMessages()[0].GetId())
	assert.Equal(t, "planet", res.Msg.GetMessages()[1].GetText())
}

func TestGetHistoryIsNeverCached(t *testing.T) {
	assert.Equal(t, "no-store", getHistory(t, stubUseCase{}).Header().Get("Cache-Control"))
}
