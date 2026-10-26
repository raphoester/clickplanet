package get_schedule_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	rulesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/rules/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/rulesv1controller/get_schedule_handler"
)

type stubQuery struct {
	answer *rulesv1.GetScheduleResponse
	err    error
}

func (s stubQuery) Schedule(context.Context) (*rulesv1.GetScheduleResponse, error) {
	return s.answer, s.err
}

func TestTheScheduleIsTheQuerys(t *testing.T) {
	answer := &rulesv1.GetScheduleResponse{Playing: &rulesv1.Playing{RuleSet: "rush"}}

	res, err := get_schedule_handler.New(stubQuery{answer: answer}).
		GetSchedule(t.Context(), connect.NewRequest(&rulesv1.GetScheduleRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(answer, res.Msg))
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	down := errors.New("postgres is down")

	_, err := get_schedule_handler.New(stubQuery{err: down}).
		GetSchedule(t.Context(), connect.NewRequest(&rulesv1.GetScheduleRequest{}))

	require.ErrorIs(t, err, down)
}
