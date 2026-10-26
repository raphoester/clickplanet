package get_schedule_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	metagamev1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/get_schedule_handler"
)

type stubQuery struct {
	answer *metagamev1.GetScheduleResponse
	err    error
}

func (s stubQuery) Schedule(context.Context) (*metagamev1.GetScheduleResponse, error) {
	return s.answer, s.err
}

func TestTheScheduleIsTheQuerys(t *testing.T) {
	answer := &metagamev1.GetScheduleResponse{Playing: &metagamev1.Playing{RuleSet: "rush"}}

	res, err := get_schedule_handler.New(stubQuery{answer: answer}).
		GetSchedule(t.Context(), connect.NewRequest(&metagamev1.GetScheduleRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(answer, res.Msg))
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	down := errors.New("postgres is down")

	_, err := get_schedule_handler.New(stubQuery{err: down}).
		GetSchedule(t.Context(), connect.NewRequest(&metagamev1.GetScheduleRequest{}))

	require.ErrorIs(t, err, down)
}
