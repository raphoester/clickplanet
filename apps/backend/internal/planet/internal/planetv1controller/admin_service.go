package planetv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/ban_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/find_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/inspect_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/reassign_country_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/revert_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/top_players_handler"
)

// AdminService is ClickService's counterpart for the loopback admin listener: handlers in a bag.
type AdminService struct {
	reassign_country_handler.ReassignCountryHandler
	find_players_handler.FindPlayersHandler
	top_players_handler.TopPlayersHandler
	ban_player_handler.BanPlayerHandler
	revert_player_handler.RevertPlayerHandler
	inspect_player_handler.InspectPlayerHandler
}

var _ planetv1connect.AdminServiceHandler = AdminService{}
