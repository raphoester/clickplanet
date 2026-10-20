package get_message_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_message_handler/message_query"
)

type Query interface {
	Message(ctx context.Context, id string) (*chatv1.GetMessageResponse, error)
}

func New(query Query) GetMessageHandler {
	return GetMessageHandler{query: query}
}

type GetMessageHandler struct {
	query Query
}

func (h GetMessageHandler) GetMessage(
	ctx context.Context,
	req *connect.Request[chatv1.GetMessageRequest],
) (*connect.Response[chatv1.GetMessageResponse], error) {
	message, err := h.query.Message(ctx, req.Msg.GetMessageId())
	if errors.Is(err, message_query.ErrNoMessage) {
		return nil, connect.NewError(connect.CodeNotFound, message_query.ErrNoMessage)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the message: %w", err)
	}

	res := connect.NewResponse(message)
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
