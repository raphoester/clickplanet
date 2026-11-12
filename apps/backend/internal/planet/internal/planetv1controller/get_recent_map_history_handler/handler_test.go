package get_recent_map_history_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_recent_map_history_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var next = time.Date(2026, 11, 11, 13, 0, 0, 0, time.UTC)

type stubQuery struct {
	history *planetv1.MapHistory
}

func (s stubQuery) RecentHistory(context.Context) (*planetv1.MapHistory, time.Time, error) {
	return s.history, next, nil
}

func ask(t *testing.T, query stubQuery, now time.Time) (*connect.Response[planetv1.GetRecentMapHistoryResponse], error) {
	t.Helper()

	return get_recent_map_history_handler.New(query, cptime.NewFixedClock(now)).GetRecentMapHistory(t.Context(), //nolint:wrapcheck // the test reads the handler's own error.
		connect.NewRequest(&planetv1.GetRecentMapHistoryRequest{}))
}

func TestTheRecentHistoryIsTheQuerysAndIsCachedUntilTheNextStep(t *testing.T) {
	query := stubQuery{history: &planetv1.MapHistory{Frames: []*planetv1.MapFrame{{
		At: timestamppb.New(next.Add(-24*time.Hour - 30*time.Minute)), Owners: []*planetv1.TileOwner{{Tile: 1, CountryId: "fr"}},
	}}}}

	res, err := ask(t, query, next.Add(-12*time.Minute-34*time.Second-500*time.Millisecond))

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.history, res.Msg.GetHistory()))
	assert.Equal(t, "public, max-age=754, s-maxage=754", res.Header().Get("Cache-Control"))
}

func TestAnAnswerPastItsStepIsNotKeptAtAll(t *testing.T) {
	res, err := ask(t, stubQuery{history: &planetv1.MapHistory{}}, next.Add(time.Second))

	require.NoError(t, err)
	assert.Equal(t, "public, max-age=0, s-maxage=0", res.Header().Get("Cache-Control"))
}
