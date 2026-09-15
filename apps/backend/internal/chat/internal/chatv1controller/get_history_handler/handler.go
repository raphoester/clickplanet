// Package get_history_handler serves chat.v1.ChatService/GetHistory.
package get_history_handler

import (
	"context"

	"connectrpc.com/connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type UseCase interface {
	Execute(ctx context.Context) []messages.Message
}

func New(useCase UseCase) GetHistoryHandler {
	return GetHistoryHandler{useCase: useCase}
}

type GetHistoryHandler struct {
	useCase UseCase
}

// GetHistory answers no-store: a cached answer would show a joiner a chat missing its last minutes.
func (h GetHistoryHandler) GetHistory(
	ctx context.Context,
	_ *connect.Request[chatv1.GetHistoryRequest],
) (*connect.Response[chatv1.GetHistoryResponse], error) {
	history := h.useCase.Execute(ctx)

	response := &chatv1.GetHistoryResponse{
		Messages: make([]*chatv1.ChatMessage, 0, len(history)),
	}
	for _, message := range history {
		response.Messages = append(response.Messages, chatmessage.Encode(message))
	}

	res := connect.NewResponse(response)
	res.Header().Set("Cache-Control", "no-store")

	return res, nil
}
