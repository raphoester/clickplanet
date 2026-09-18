// Package react_handler serves chat.v1.ChatService/React.
package react_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/usecases/react_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type UseCase interface {
	Execute(ctx context.Context, in react_usecase.In) ([]reactions.Count, error)
}

func New(useCase UseCase) ReactHandler {
	return ReactHandler{useCase: useCase}
}

type ReactHandler struct {
	useCase UseCase
}

func (h ReactHandler) React(
	ctx context.Context,
	req *connect.Request[chatv1.ReactRequest],
) (*connect.Response[chatv1.ReactResponse], error) {
	reaction, err := chatmessage.Reaction(req.Msg.GetReaction())
	if err != nil {
		return nil, toConnect(err)
	}

	counts, err := h.useCase.Execute(ctx, react_usecase.In{
		Account:   messages.AccountIDOf(cpctx.GetAccount(ctx)),
		MessageID: messages.MessageID(req.Msg.GetMessageId()),
		Reaction:  reaction,
		On:        req.Msg.GetOn(),
	})
	if err != nil {
		return nil, toConnect(err)
	}

	return connect.NewResponse(&chatv1.ReactResponse{Reactions: chatmessage.EncodeCounts(counts)}), nil
}

// toConnect sends the bare sentinel, as SendMessage does. Anything else is left for the error net.
func toConnect(err error) error {
	switch {
	case errors.Is(err, reactions.ErrInvalidReaction):
		return connect.NewError(connect.CodeInvalidArgument, reactions.ErrInvalidReaction)
	case errors.Is(err, reactions.ErrUnknownMessage):
		return connect.NewError(connect.CodeNotFound, reactions.ErrUnknownMessage)
	case errors.Is(err, messages.ErrAuthorUnavailable):
		return connect.NewError(connect.CodeUnavailable, messages.ErrAuthorUnavailable)
	default:
		return err
	}
}
