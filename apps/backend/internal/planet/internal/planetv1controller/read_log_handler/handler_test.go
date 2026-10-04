package read_log_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/read_log_handler"
)

type asked struct {
	from  uint64
	limit uint32
}

type stubQuery struct {
	answer *planetv1.ReadLogResponse
	err    error
	asked  []asked
}

func (s *stubQuery) Entries(_ context.Context, from uint64, limit uint32) (*planetv1.ReadLogResponse, error) {
	s.asked = append(s.asked, asked{from: from, limit: limit})
	return s.answer, s.err
}

func TestTheEntriesAreTheQuerysFromThePositionAndLimitAsked(t *testing.T) {
	query := &stubQuery{answer: &planetv1.ReadLogResponse{Entries: []*planetv1.LogEntry{
		{Position: 42, Fact: &planetv1.LogEntry_Take{Take: &planetv1.Take{TileId: 7, Country: "fr"}}},
	}}}

	res, err := read_log_handler.New(query).ReadLog(t.Context(),
		connect.NewRequest(&planetv1.ReadLogRequest{FromPosition: 42, Limit: 10}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	assert.Equal(t, []asked{{from: 42, limit: 10}}, query.asked)
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"), "a take's reverted flag can change")
}

func TestAFailedReadIsTheQuerysError(t *testing.T) {
	failed := errors.New("postgres is down")

	_, err := read_log_handler.New(&stubQuery{err: failed}).ReadLog(t.Context(),
		connect.NewRequest(&planetv1.ReadLogRequest{}))

	assert.ErrorIs(t, err, failed)
}
