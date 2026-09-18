// Package get_history_handler serves chat.v1.ChatService/GetHistory.
package get_history_handler

import (
	"context"

	"connectrpc.com/connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/get_history_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type UseCase interface {
	Execute(ctx context.Context, account messages.AccountID) ([]get_history_usecase.Entry, error)
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
	history, err := h.useCase.Execute(ctx, messages.AccountIDOf(cpctx.GetAccount(ctx)))
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}

	response := &chatv1.GetHistoryResponse{
		Messages: make([]*chatv1.ChatMessage, 0, len(history)),
	}
	for _, entry := range history {
		response.Messages = append(response.Messages, chatmessage.Encode(entry.Message, entry.Reactions, entry.ReactionsVersion))
	}

	res := connect.NewResponse(response)
	res.Header().Set("Cache-Control", "no-store")

	return res, nil
}
