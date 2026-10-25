package seasonsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_hall_of_fame_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_my_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_race_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_standings_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/listen_for_events_handler"
)

type SeasonService struct {
	get_season_handler.GetSeasonHandler
	get_standings_handler.GetStandingsHandler
	get_my_season_handler.GetMySeasonHandler
	listen_for_events_handler.ListenForEventsHandler
	get_race_handler.GetRaceHandler
	get_hall_of_fame_handler.GetHallOfFameHandler
}

var _ seasonsv1connect.SeasonServiceHandler = SeasonService{}
