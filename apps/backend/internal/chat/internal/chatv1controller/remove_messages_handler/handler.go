package remove_messages_handler

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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/usecases/remove_messages_usecase"
)

var ErrNoMessageID = errors.New("no message id")

func New(useCase remove_messages_usecase.Executor) RemoveMessagesHandler {
	return RemoveMessagesHandler{useCase: useCase}
}

type RemoveMessagesHandler struct {
	useCase remove_messages_usecase.Executor
}

// A refusal names the message it is about, so the caller can send the others again without it.
func (h RemoveMessagesHandler) RemoveMessages(
	ctx context.Context,
	req *connect.Request[chatv1.RemoveMessagesRequest],
) (*connect.Response[chatv1.RemoveMessagesResponse], error) {
	asked := make([]remove_messages_usecase.Removal, 0, len(req.Msg.GetRemovals()))
	for i, removal := range req.Msg.GetRemovals() {
		if removal.GetMessageId() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("removal %d: %w", i, ErrNoMessageID))
		}
		asked = append(asked, remove_messages_usecase.Removal{
			Message: messages.MessageID(removal.GetMessageId()), Reason: removal.GetReason(),
		})
	}

	removedAt, err := h.useCase.Execute(ctx, remove_messages_usecase.In{Removals: asked, By: chatoperator.Of(ctx)})
	if errors.Is(err, removals.ErrUnknownMessage) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if errors.Is(err, moderation.ErrInvalidReason) ||
		errors.Is(err, remove_messages_usecase.ErrNoRemoval) ||
		errors.Is(err, remove_messages_usecase.ErrTooManyRemovals) ||
		errors.Is(err, remove_messages_usecase.ErrRemovedTwice) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to remove the messages: %w", err)
	}

	return connect.NewResponse(&chatv1.RemoveMessagesResponse{RemovedAt: timestamppb.New(removedAt)}), nil
}
