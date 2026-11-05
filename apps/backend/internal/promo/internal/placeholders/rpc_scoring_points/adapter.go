package rpc_scoring_points

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/placeholders"
)

type Scoring interface {
	GetRankPoints(context.Context, *connect.Request[scoringv1.GetRankPointsRequest]) (*connect.Response[scoringv1.GetRankPointsResponse], error)
}

func New(scoring Scoring) *Adapter {
	return &Adapter{scoring: scoring}
}

// A battle's points, from scoring, which keeps the one table.
type Adapter struct {
	scoring Scoring
}

func (a *Adapter) Battle(ctx context.Context, times uint32) (placeholders.Battle, error) {
	res, err := a.scoring.GetRankPoints(ctx, connect.NewRequest(&scoringv1.GetRankPointsRequest{Times: times}))
	if err != nil {
		return placeholders.Battle{}, fmt.Errorf("failed to ask scoring what each rank of a round of %d times scores: %w", times, err)
	}
	points := make(map[uint32]uint32, len(res.Msg.GetRanks()))
	for _, rank := range res.Msg.GetRanks() {
		points[rank.GetRank()] = rank.GetPoints()
	}
	return placeholders.Battle{Times: times, Points: points}, nil
}
