package chatv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/ban_member_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_bans_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/unban_member_handler"
)

// AdminService is ChatService's counterpart for the loopback admin listener: handlers in a bag.
type AdminService struct {
	ban_member_handler.BanMemberHandler
	unban_member_handler.UnbanMemberHandler
	list_bans_handler.ListBansHandler
}

var _ chatv1connect.AdminServiceHandler = AdminService{}
