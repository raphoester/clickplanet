package get_rank_points_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_rank_points_handler"
)

func TestTheRanksOfARoundOfTwoTimesScoreTwiceADays(t *testing.T) {
	res, err := get_rank_points_handler.New().
		GetRankPoints(t.Context(), connect.NewRequest(&scoringv1.GetRankPointsRequest{Times: 2}))

	require.NoError(t, err)
	points := make([]uint32, 0, len(res.Msg.GetRanks()))
	for i, rank := range res.Msg.GetRanks() {
		assert.Equal(t, uint32(i+1), rank.GetRank()) //nolint:gosec // ten ranks.
		points = append(points, rank.GetPoints())
	}
	assert.Equal(t, []uint32{50, 36, 30, 24, 20, 16, 12, 8, 4, 2}, points)
}

func TestNoTimesIsInvalid(t *testing.T) {
	_, err := get_rank_points_handler.New().
		GetRankPoints(t.Context(), connect.NewRequest(&scoringv1.GetRankPointsRequest{}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
