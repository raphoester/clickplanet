package chatv1controller

import (
	"context"

	"connectrpc.com/connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/domain/chat_service"
)

type ChatService struct {
	chatService chat_service.IService
}

var _ chatv1connect.ChatServiceHandler = (*ChatService)(nil)

func NewChatService(chatService chat_service.IService) *ChatService {
	return &ChatService{chatService: chatService}
}

func (s *ChatService) SendMessage(
	ctx context.Context,
	req *connect.Request[chatv1.SendMessageRequest],
) (*connect.Response[chatv1.SendMessageResponse], error) {
	message, err := s.chatService.Post(ctx, chat_service.PostRequest{
		AuthorName: req.Msg.GetAuthorName(),
		AuthorID:   req.Msg.GetAuthorId(),
		CountryID:  req.Msg.GetCountryId(),
		Text:       req.Msg.GetText(),
		UserAgent:  req.Header().Get("User-Agent"),
	})

	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&chatv1.SendMessageResponse{
		Message: toProto(message),
	}), nil
}

func (s *ChatService) GetHistory(
	ctx context.Context,
	_ *connect.Request[chatv1.GetHistoryRequest],
) (*connect.Response[chatv1.GetHistoryResponse], error) {
	messages := s.chatService.History(ctx)

	response := &chatv1.GetHistoryResponse{
		Messages: make([]*chatv1.ChatMessage, 0, len(messages)),
	}
	for _, message := range messages {
		response.Messages = append(response.Messages, toProto(message))
	}

	res := connect.NewResponse(response)
	res.Header().Set("Cache-Control", "no-store")

	return res, nil
}

func toProto(message domain.ChatMessage) *chatv1.ChatMessage {
	return &chatv1.ChatMessage{
		Id:           message.ID,
		SentAtUnixMs: message.SentAt.UnixMilli(),
		AuthorName:   message.AuthorName,
		AuthorTag:    message.AuthorTag,
		CountryId:    message.CountryID,
		Text:         message.Text,
	}
}
