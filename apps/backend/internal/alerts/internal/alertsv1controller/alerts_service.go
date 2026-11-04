package alertsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/alerts/v1/alertsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/get_discord_link_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/get_player_link_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/link_members_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/unlink_player_handler"
)

type AlertsService struct {
	get_discord_link_handler.GetDiscordLinkHandler
}

var _ alertsv1connect.AlertsServiceHandler = AlertsService{}

type AdminService struct {
	get_player_link_handler.GetPlayerLinkHandler
	unlink_player_handler.UnlinkPlayerHandler
	link_members_handler.LinkMembersHandler
}

var _ alertsv1connect.AdminServiceHandler = AdminService{}
