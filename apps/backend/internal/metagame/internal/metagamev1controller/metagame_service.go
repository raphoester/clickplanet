package metagamev1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1/metagamev1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/get_turn_handler"
)

type MetagameService struct {
	get_turn_handler.GetTurnHandler
}

var _ metagamev1connect.MetagameServiceHandler = MetagameService{}
