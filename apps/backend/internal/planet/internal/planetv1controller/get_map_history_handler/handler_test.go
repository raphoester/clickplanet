package get_map_history_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/forgetting"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_map_history_handler"
)

type countingQuery struct {
	read int
}

func (q *countingQuery) History(context.Context, time.Time, time.Time, time.Duration) (*planetv1.MapHistory, error) {
	q.read++
	return &planetv1.MapHistory{Frames: []*planetv1.MapFrame{{}}}, nil
}

type fixedForgetting forgetting.Forgetting

func (f fixedForgetting) Forgetting(context.Context) (forgetting.Forgetting, error) {
	return forgetting.Forgetting(f), nil
}

func ask(t *testing.T, handler get_map_history_handler.GetMapHistoryHandler, kept *string) *planetv1.GetMapHistoryResponse {
	t.Helper()
	since := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	res, err := handler.GetMapHistory(t.Context(), connect.NewRequest(&planetv1.GetMapHistoryRequest{
		Since: timestamppb.New(since), Until: timestamppb.New(since.Add(time.Hour)), Step: durationpb.New(time.Minute),
		KeptForgetting: kept,
	}))
	require.NoError(t, err)
	return res.Msg
}

func TestAHistoryKeptIsAnsweredKeptUntilARevertForgetsMore(t *testing.T) {
	query := &countingQuery{}
	before, after := "a1", "b2"

	first := ask(t, get_map_history_handler.New(query, fixedForgetting(before)), nil)
	kept := ask(t, get_map_history_handler.New(query, fixedForgetting(before)), &before)
	reverted := ask(t, get_map_history_handler.New(query, fixedForgetting(after)), &before)

	assert.True(t, proto.Equal(&planetv1.GetMapHistoryResponse{
		History: &planetv1.MapHistory{Frames: []*planetv1.MapFrame{{}}}, Forgetting: before,
	}, first))
	assert.True(t, proto.Equal(&planetv1.GetMapHistoryResponse{Forgetting: before, Kept: true}, kept), "no history read")
	assert.Equal(t, after, reverted.GetForgetting())
	assert.False(t, reverted.GetKept())
	assert.NotNil(t, reverted.GetHistory())
	assert.Equal(t, 2, query.read)
}
