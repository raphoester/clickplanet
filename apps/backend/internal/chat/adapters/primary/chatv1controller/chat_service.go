package chatv1controller

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/domain/chat_service"
)

type MessagesSubscriber interface {
	Subscribe(ctx context.Context) (<-chan domain.ChatMessage, error)
}

type ChatService struct {
	chatService chat_service.IService
	subscriber  MessagesSubscriber
	heartbeat   time.Duration
}

var _ chatv1connect.ChatServiceHandler = (*ChatService)(nil)

func NewChatService(
	chatService chat_service.IService,
	subscriber MessagesSubscriber,
	heartbeat time.Duration,
) *ChatService {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}

	return &ChatService{chatService: chatService, subscriber: subscriber, heartbeat: heartbeat}
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

// The request context is what unsubscribes, and it is cancelled however the stream ends.
func (s *ChatService) ListenForEvents(
	ctx context.Context,
	_ *connect.Request[chatv1.ListenForEventsRequest],
	stream *connect.ServerStream[chatv1.ChatEvent],
) error {
	messages, err := s.subscriber.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to chat messages: %w", err)
	}

	heartbeat := time.NewTicker(s.heartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-heartbeat.C:
			if err := stream.Send(heartbeatEvent()); err != nil {
				return err
			}

		case message, open := <-messages:
			if !open {
				return nil
			}

			if err := stream.Send(messageEvent(message)); err != nil {
				return err
			}
		}
	}
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
