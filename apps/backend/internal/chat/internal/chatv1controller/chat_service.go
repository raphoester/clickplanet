package chatv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_history_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_message_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_mute_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_raid_mode_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_messages_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_moderation_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/mark_seen_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/mute_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/react_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/remove_message_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/send_message_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/set_raid_mode_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/unmute_handler"
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
	unmute_handler.UnmuteHandler
	get_mute_handler.GetMuteHandler
	remove_message_handler.RemoveMessageHandler
	get_message_handler.GetMessageHandler
	list_messages_handler.ListMessagesHandler
	list_moderation_handler.ListModerationHandler
	set_raid_mode_handler.SetRaidModeHandler
	get_raid_mode_handler.GetRaidModeHandler
}

var _ chatv1connect.AdminServiceHandler = AdminService{}
