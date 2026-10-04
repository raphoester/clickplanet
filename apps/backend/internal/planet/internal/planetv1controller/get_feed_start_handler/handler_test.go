package get_feed_start_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_feed_start_handler"
)

type stubQuery struct {
	answer *planetv1.GetFeedStartResponse
	err    error
}

func (s stubQuery) Start(context.Context) (*planetv1.GetFeedStartResponse, error) {
	return s.answer, s.err
}

func TestTheStartIsTheQuerys(t *testing.T) {
	res, err := get_feed_start_handler.New(stubQuery{answer: &planetv1.GetFeedStartResponse{Position: 42}}).
		GetFeedStart(t.Context(), connect.NewRequest(&planetv1.GetFeedStartRequest{}))

	require.NoError(t, err)
	assert.Equal(t, uint64(42), res.Msg.GetPosition())
}

func TestAFailedReadIsTheQuerysError(t *testing.T) {
	failed := errors.New("postgres is down")

	_, err := get_feed_start_handler.New(stubQuery{err: failed}).
		GetFeedStart(t.Context(), connect.NewRequest(&planetv1.GetFeedStartRequest{}))

	assert.ErrorIs(t, err, failed)
}
