package scoringv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_board_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_my_tally_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_round_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_scores_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/listen_for_events_handler"
)

type ScoringService struct {
	get_board_handler.GetBoardHandler
	get_my_tally_handler.GetMyTallyHandler
	get_round_handler.GetRoundHandler
	get_scores_handler.GetScoresHandler
	listen_for_events_handler.ListenForEventsHandler
}

var _ scoringv1connect.ScoringServiceHandler = ScoringService{}
