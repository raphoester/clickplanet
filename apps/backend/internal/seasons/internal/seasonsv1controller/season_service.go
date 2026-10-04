package seasonsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_my_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_standings_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/rebuild_standings_handler"
)

type SeasonService struct {
	get_season_handler.GetSeasonHandler
	get_standings_handler.GetStandingsHandler
	get_my_season_handler.GetMySeasonHandler
}

var _ seasonsv1connect.SeasonServiceHandler = SeasonService{}

type AdminService struct {
	rebuild_standings_handler.RebuildStandingsHandler
}

var _ seasonsv1connect.AdminServiceHandler = AdminService{}
