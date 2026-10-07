package chatv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_history_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/mark_seen_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/mute_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/react_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/send_message_handler"
)

type ChatService struct {
	send_message_handler.SendMessageHandler
	get_history_handler.GetHistoryHandler
	listen_for_events_handler.ListenForEventsHandler
	react_handler.ReactHandler
	mark_seen_handler.MarkSeenHandler
}

var _ chatv1connect.ChatServiceHandler = ChatService{}

type AdminService struct {
	mute_handler.MuteHandler
}

var _ chatv1connect.AdminServiceHandler = AdminService{}
