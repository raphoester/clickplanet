package playerv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/admin_get_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/announce_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_accounts_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_profile_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_roster_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/introduce_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/leave_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/list_renames_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/rename_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/search_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/set_color_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/set_name_handler"
)

type PlayerService struct {
	get_profile_handler.GetProfileHandler
	set_name_handler.SetNameHandler
	set_color_handler.SetColorHandler
	announce_handler.AnnounceHandler
	leave_handler.LeaveHandler
	get_roster_handler.GetRosterHandler
	listen_for_events_handler.ListenForEventsHandler
	get_player_handler.GetPlayerHandler
}

var _ playerv1connect.PlayerServiceHandler = PlayerService{}

type InternalService struct {
	introduce_player_handler.IntroducePlayerHandler
	get_players_handler.GetPlayersHandler
	get_accounts_handler.GetAccountsHandler
}

var _ playerv1connect.InternalServiceHandler = InternalService{}

type AdminService struct {
	get_players_handler.GetPlayersHandler
	admin_get_player_handler.AdminGetPlayerHandler
	get_roster_handler.GetRosterHandler
	search_players_handler.SearchPlayersHandler
	rename_player_handler.RenamePlayerHandler
	list_renames_handler.ListRenamesHandler
}

var _ playerv1connect.AdminServiceHandler = AdminService{}
