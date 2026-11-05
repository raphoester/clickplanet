package get_rank_points_handler

import (
	"context"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

func New() GetRankPointsHandler {
	return GetRankPointsHandler{}
}

// GetRankPointsHandler answers the rule itself, books.RankPointsOf: nothing is kept for it.
type GetRankPointsHandler struct{}

func (GetRankPointsHandler) GetRankPoints(
	_ context.Context,
	req *connect.Request[scoringv1.GetRankPointsRequest],
) (*connect.Response[scoringv1.GetRankPointsResponse], error) {
	ranks, err := books.RankPointsOf(req.Msg.GetTimes())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := &scoringv1.GetRankPointsResponse{Ranks: make([]*scoringv1.RankPoints, 0, len(ranks))}
	for _, rank := range ranks {
		out.Ranks = append(out.Ranks, &scoringv1.RankPoints{Rank: rank.Rank, Points: rank.Points})
	}
	return connect.NewResponse(out), nil
}
