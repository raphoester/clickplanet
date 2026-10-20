package remove_message_handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals"
)

var ErrNoMessageID = errors.New("no message id")

type UseCase interface {
	Execute(ctx context.Context, id messages.MessageID) (time.Time, error)
}

func New(useCase UseCase) RemoveMessageHandler {
	return RemoveMessageHandler{useCase: useCase}
}

type RemoveMessageHandler struct {
	useCase UseCase
}

func (h RemoveMessageHandler) RemoveMessage(
	ctx context.Context,
	req *connect.Request[chatv1.RemoveMessageRequest],
) (*connect.Response[chatv1.RemoveMessageResponse], error) {
	id := req.Msg.GetMessageId()
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrNoMessageID)
	}

	removedAt, err := h.useCase.Execute(ctx, messages.MessageID(id))
	if errors.Is(err, removals.ErrUnknownMessage) {
		return nil, connect.NewError(connect.CodeNotFound, removals.ErrUnknownMessage)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to remove the message: %w", err)
	}

	return connect.NewResponse(&chatv1.RemoveMessageResponse{RemovedAt: timestamppb.New(removedAt)}), nil
}
