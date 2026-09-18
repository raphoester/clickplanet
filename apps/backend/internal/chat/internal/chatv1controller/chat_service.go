package chatv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_history_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/react_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/send_message_handler"
)

// ChatService is the four handlers in a bag, for the generated handler.
type ChatService struct {
	send_message_handler.SendMessageHandler
	get_history_handler.GetHistoryHandler
	listen_for_events_handler.ListenForEventsHandler
	react_handler.ReactHandler
}

var _ chatv1connect.ChatServiceHandler = ChatService{}
