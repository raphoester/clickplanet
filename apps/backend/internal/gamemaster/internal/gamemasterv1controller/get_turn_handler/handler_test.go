package get_turn_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_turn_handler"
)

type stubQuery struct {
	now *gamemasterv1.GetTurnResponse
	err error
}

func (s stubQuery) Now(context.Context) (*gamemasterv1.GetTurnResponse, error) {
	return s.now, s.err
}

func TestTheTurnAndTheTimelineAreTheQuerys(t *testing.T) {
	now := &gamemasterv1.GetTurnResponse{
		Turn:     &gamemasterv1.Turn{Kind: gamemasterv1.EntryKind_ENTRY_KIND_BATTLE, Description: "Triple points."},
		Timeline: &gamemasterv1.Timeline{Battles: []*gamemasterv1.Battle{{EntryId: "next", Description: "Double points."}}},
	}

	res, err := get_turn_handler.New(stubQuery{now: now}).
		GetTurn(t.Context(), connect.NewRequest(&gamemasterv1.GetTurnRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(now, res.Msg))
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	down := errors.New("postgres is down")

	_, err := get_turn_handler.New(stubQuery{err: down}).
		GetTurn(t.Context(), connect.NewRequest(&gamemasterv1.GetTurnRequest{}))

	require.ErrorIs(t, err, down)
}
