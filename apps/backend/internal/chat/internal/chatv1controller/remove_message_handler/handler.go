package remove_message_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatoperator"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/usecases/remove_message_usecase"
)

var ErrNoMessageID = errors.New("no message id")

func New(useCase remove_message_usecase.Executor) RemoveMessageHandler {
	return RemoveMessageHandler{useCase: useCase}
}

type RemoveMessageHandler struct {
	useCase remove_message_usecase.Executor
}

func (h RemoveMessageHandler) RemoveMessage(
	ctx context.Context,
	req *connect.Request[chatv1.RemoveMessageRequest],
) (*connect.Response[chatv1.RemoveMessageResponse], error) {
	id := req.Msg.GetMessageId()
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrNoMessageID)
	}

	removedAt, err := h.useCase.Execute(ctx, remove_message_usecase.In{
		Message: messages.MessageID(id), Reason: req.Msg.GetReason(), By: chatoperator.Of(ctx),
	})
	if errors.Is(err, removals.ErrUnknownMessage) {
		return nil, connect.NewError(connect.CodeNotFound, removals.ErrUnknownMessage)
	}
	if errors.Is(err, moderation.ErrInvalidReason) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to remove the message: %w", err)
	}

	return connect.NewResponse(&chatv1.RemoveMessageResponse{RemovedAt: timestamppb.New(removedAt)}), nil
}
