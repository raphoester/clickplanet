package gamemasterv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1/gamemasterv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_battle_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_timeline_handler"
)

// What another module asks the gamemaster, on the internal listener.
type InternalService struct {
	get_battle_handler.GetBattleHandler
	get_timeline_handler.GetTimelineHandler
}

var _ gamemasterv1connect.InternalServiceHandler = InternalService{}
