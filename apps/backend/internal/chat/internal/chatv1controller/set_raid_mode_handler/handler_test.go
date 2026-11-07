package set_raid_mode_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/set_raid_mode_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids/inmemory_raid_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids/usecases/set_raid_mode_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 11, 7, 20, 0, 0, 0, time.UTC)

type recording struct {
	set_raid_mode_usecase.Executor

	in set_raid_mode_usecase.In
}

func (r *recording) Execute(ctx context.Context, in set_raid_mode_usecase.In) (raids.RaidMode, bool, error) {
	r.in = in
	return r.Executor.Execute(ctx, in)
}

func handler() (set_raid_mode_handler.SetRaidModeHandler, *recording) {
	inner := &recording{Executor: set_raid_mode_usecase.New(inmemory_raid_storage.New(), cptime.NewFixedClock(now))}
	return set_raid_mode_handler.New(inner), inner
}

func TestARaidModeSetIsAnsweredWithItsEndAndTakesWhoSetIt(t *testing.T) {
	h, inner := handler()
	ctx := cpctx.AddOperatorToContext(t.Context(), cpctx.OperatorOf(messages.AccountID{15: 9}, "Claude"))

	res, err := h.SetRaidMode(ctx, connect.NewRequest(&chatv1.SetRaidModeRequest{
		Duration: durationpb.New(time.Hour), Reason: "a raid of guests",
	}))

	require.NoError(t, err)
	assert.Equal(t, timestamppb.New(now), res.Msg.GetRaidMode().GetStartedAt())
	assert.Equal(t, timestamppb.New(now.Add(time.Hour)), res.Msg.GetRaidMode().GetEndsAt())
	assert.Equal(t, "a raid of guests", res.Msg.GetRaidMode().GetReason())
	assert.Equal(t, "Claude", inner.in.By.Client())
}

func TestNoDurationEndsItAndAnswersNone(t *testing.T) {
	h, _ := handler()

	res, err := h.SetRaidMode(t.Context(), connect.NewRequest(&chatv1.SetRaidModeRequest{}))

	require.NoError(t, err)
	assert.Nil(t, res.Msg.GetRaidMode())
}

func TestADurationPastADayIsAnInvalidArgument(t *testing.T) {
	h, _ := handler()

	_, err := h.SetRaidMode(t.Context(), connect.NewRequest(&chatv1.SetRaidModeRequest{
		Duration: durationpb.New(48 * time.Hour),
	}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
