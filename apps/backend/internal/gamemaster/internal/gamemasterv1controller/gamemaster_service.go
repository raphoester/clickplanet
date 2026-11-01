package gamemasterv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1/gamemasterv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_season_replay_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_season_totals_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_turn_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/listen_for_events_handler"
)

type GamemasterService struct {
	get_turn_handler.GetTurnHandler
	listen_for_events_handler.ListenForEventsHandler
	get_season_totals_handler.GetSeasonTotalsHandler
	get_season_replay_handler.GetSeasonReplayHandler
}

var _ gamemasterv1connect.GamemasterServiceHandler = GamemasterService{}
