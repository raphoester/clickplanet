package alertsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/alerts/v1/alertsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/complete_discord_link_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/get_discord_link_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/start_discord_link_handler"
)

type AlertsService struct {
	get_discord_link_handler.GetDiscordLinkHandler
	start_discord_link_handler.StartDiscordLinkHandler
	complete_discord_link_handler.CompleteDiscordLinkHandler
}

var _ alertsv1connect.AlertsServiceHandler = AlertsService{}
