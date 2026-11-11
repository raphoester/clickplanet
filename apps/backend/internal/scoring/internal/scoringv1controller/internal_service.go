package scoringv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/close_book_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/close_round_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_rank_points_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_round_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_round_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_table_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/open_book_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/open_round_handler"
)

type InternalService struct {
	open_book_handler.OpenBookHandler
	open_round_handler.OpenRoundHandler
	close_round_handler.CloseRoundHandler
	close_book_handler.CloseBookHandler
	get_round_handler.GetRoundHandler
	get_table_handler.GetTableHandler
	get_round_players_handler.GetRoundPlayersHandler
	get_rank_points_handler.GetRankPointsHandler
}

var _ scoringv1connect.InternalServiceHandler = InternalService{}
