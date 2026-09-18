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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/get_history_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
)

type stubUseCase []get_history_usecase.Entry

func (s stubUseCase) Execute(context.Context, messages.AccountID) ([]get_history_usecase.Entry, error) {
	return s, nil
}

func getHistory(t *testing.T, useCase stubUseCase) *connect.Response[chatv1.GetHistoryResponse] {
	t.Helper()

	res, err := get_history_handler.New(useCase).
		GetHistory(t.Context(), connect.NewRequest(&chatv1.GetHistoryRequest{}))
	require.NoError(t, err)

	return res
}

func TestGetHistoryMapsEveryMessageInOrder(t *testing.T) {
	res := getHistory(t, stubUseCase{
		{Message: messages.Message{ID: "message-1", Text: "hello"}},
		{Message: messages.Message{ID: "message-2", Text: "planet"}},
	})

	require.Len(t, res.Msg.GetMessages(), 2)
	assert.Equal(t, "message-1", res.Msg.GetMessages()[0].GetId())
	assert.Equal(t, "planet", res.Msg.GetMessages()[1].GetText())
}

func TestGetHistoryMapsTheReactions(t *testing.T) {
	res := getHistory(t, stubUseCase{{Message: messages.Message{ID: "message-1"}, Reactions: []reactions.Count{
		{Reaction: reactions.Reaction(chatv1.Reaction_REACTION_CLOWN), Count: 3, Mine: true},
	}}})

	reactions := res.Msg.GetMessages()[0].GetReactions()
	require.Len(t, reactions, 1)
	assert.Equal(t, chatv1.Reaction_REACTION_CLOWN, reactions[0].GetReaction())
	assert.Equal(t, uint32(3), reactions[0].GetCount())
	assert.True(t, reactions[0].GetMine())
}

func TestGetHistoryIsNeverCached(t *testing.T) {
	assert.Equal(t, "no-store", getHistory(t, stubUseCase{}).Header().Get("Cache-Control"))
}
