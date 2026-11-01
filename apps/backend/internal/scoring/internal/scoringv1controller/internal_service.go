package scoringv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_board_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_round_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_round_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_table_handler"
)

type InternalService struct {
	get_round_handler.GetRoundHandler
	get_table_handler.GetTableHandler
	get_round_players_handler.GetRoundPlayersHandler
	get_board_handler.GetBoardHandler
}

var _ scoringv1connect.InternalServiceHandler = InternalService{}
