package gamemasterv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1/gamemasterv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_battle_handler"
)

// What another module asks the gamemaster, on the internal listener.
type InternalService struct {
	get_battle_handler.GetBattleHandler
}

var _ gamemasterv1connect.InternalServiceHandler = InternalService{}
