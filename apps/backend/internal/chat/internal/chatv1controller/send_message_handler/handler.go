// Package send_message_handler serves chat.v1.ChatService/SendMessage.
package send_message_handler

import (
	"context"

	"connectrpc.com/connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type UseCase interface {
	Execute(ctx context.Context, in send_message_usecase.In) (messages.Message, error)
}

func New(useCase UseCase) SendMessageHandler {
	return SendMessageHandler{useCase: useCase}
}

type SendMessageHandler struct {
	useCase UseCase
}

func (h SendMessageHandler) SendMessage(
	ctx context.Context,
	req *connect.Request[chatv1.SendMessageRequest],
) (*connect.Response[chatv1.SendMessageResponse], error) {
	message, err := h.useCase.Execute(ctx, send_message_usecase.In{
		Account:   messages.AccountIDOf(cpctx.GetAccount(ctx)),
		AuthorID:  req.Msg.GetAuthorId(),
		CountryID: req.Msg.GetCountryId(),
		Text:      req.Msg.GetText(),
		UserAgent: req.Header().Get("User-Agent"),
	})
	if err != nil {
		return nil, toConnect(err)
	}

	return connect.NewResponse(&chatv1.SendMessageResponse{
		Message: chatmessage.Encode(message, nil, 0),
	}), nil
}
