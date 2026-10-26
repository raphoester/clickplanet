package metagamev1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1/metagamev1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/get_turn_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/listen_for_events_handler"
)

type MetagameService struct {
	get_turn_handler.GetTurnHandler
	listen_for_events_handler.ListenForEventsHandler
}

var _ metagamev1connect.MetagameServiceHandler = MetagameService{}
