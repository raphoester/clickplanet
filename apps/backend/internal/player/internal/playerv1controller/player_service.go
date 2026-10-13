package playerv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/announce_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_author_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_authors_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_fronts_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_profile_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_roster_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_stats_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_titles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/leave_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/name_accounts_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/reconcile_titles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/set_color_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/set_name_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/wear_title_handler"
)

type PlayerService struct {
	get_profile_handler.GetProfileHandler
	set_name_handler.SetNameHandler
	set_color_handler.SetColorHandler
	get_stats_handler.GetStatsHandler
	announce_handler.AnnounceHandler
	leave_handler.LeaveHandler
	get_roster_handler.GetRosterHandler
	listen_for_events_handler.ListenForEventsHandler
	get_player_handler.GetPlayerHandler
	get_titles_handler.GetTitlesHandler
	wear_title_handler.WearTitleHandler
	get_fronts_handler.GetFrontsHandler
}

var _ playerv1connect.PlayerServiceHandler = PlayerService{}

type InternalService struct {
	get_author_handler.GetAuthorHandler
	get_authors_handler.GetAuthorsHandler
}

var _ playerv1connect.InternalServiceHandler = InternalService{}

type AdminService struct {
	reconcile_titles_handler.ReconcileTitlesHandler
	name_accounts_handler.NameAccountsHandler
}

var _ playerv1connect.AdminServiceHandler = AdminService{}
