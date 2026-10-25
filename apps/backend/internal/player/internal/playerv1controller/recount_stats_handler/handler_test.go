package recount_stats_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/recount_stats_handler"
)

type stubUseCase struct{ recounted int }

func (s stubUseCase) Execute(context.Context) (int, error) { return s.recounted, nil }

func TestTheAnswerSaysHowManyAccountsWereRecounted(t *testing.T) {
	res, err := recount_stats_handler.New(stubUseCase{recounted: 164}).RecountStats(t.Context(),
		connect.NewRequest(&playerv1.RecountStatsRequest{}))

	require.NoError(t, err)
	assert.Equal(t, uint32(164), res.Msg.GetRecounted())
}
