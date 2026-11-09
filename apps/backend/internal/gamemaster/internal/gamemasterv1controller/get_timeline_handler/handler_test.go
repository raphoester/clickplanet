package get_timeline_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_timeline_handler"
)

type turn struct {
	answer *gamemasterv1.GetTurnResponse
	err    error
}

func (t turn) Now(context.Context) (*gamemasterv1.GetTurnResponse, error) {
	return t.answer, t.err
}

func TestTheTimelineIsTheBattlesThePlayersSeeComing(t *testing.T) {
	battles := []*gamemasterv1.Battle{
		{EntryId: "01a1b1db-9bf9-7df9-917e-5d99aefe8668", Description: "Ring War", Times: 2},
		{RoundId: "season-1/finale", Times: 3, EndsSeason: true},
	}
	now := &gamemasterv1.GetTurnResponse{
		Turn: &gamemasterv1.Turn{Kind: gamemasterv1.EntryKind_ENTRY_KIND_PLAIN}, Timeline: &gamemasterv1.Timeline{Battles: battles},
	}

	answer, err := get_timeline_handler.New(turn{answer: now}).GetTimeline(t.Context(), connect.NewRequest(&gamemasterv1.GetTimelineRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(&gamemasterv1.GetTimelineResponse{Battles: battles}, answer.Msg))
}

func TestATurnThatCannotBeReadIsAnError(t *testing.T) {
	broken := errors.New("the schedule cannot be read")

	_, err := get_timeline_handler.New(turn{err: broken}).GetTimeline(t.Context(), connect.NewRequest(&gamemasterv1.GetTimelineRequest{}))

	require.ErrorIs(t, err, broken)
}
