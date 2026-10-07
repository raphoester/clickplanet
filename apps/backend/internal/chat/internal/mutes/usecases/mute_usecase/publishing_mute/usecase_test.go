package publishing_mute_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/mute_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/mute_usecase/publishing_mute"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type stubExecutor struct {
	mute mutes.Mute
	err  error
}

func (s stubExecutor) Execute(context.Context, mute_usecase.In) (mutes.Mute, error) {
	return s.mute, s.err
}

var (
	at    = time.Date(2026, 10, 7, 20, 0, 0, 0, time.UTC)
	bully = messages.AccountIDOf("0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10")
	kept  = mutes.NewMute(mutes.MuteID{15: 1}, mutes.CallerOf(bully, "2a01:e0a:1:2::/64"), at, 90*time.Minute)
)

func TestAKeptMuteIsPublishedWithItsAccountTimeAndDuration(t *testing.T) {
	events := cpbootstrap.NewRecordedEvents()

	mute, err := publishing_mute.New(stubExecutor{mute: kept}, events).Execute(t.Context(), mute_usecase.In{Account: bully})

	require.NoError(t, err)
	assert.Equal(t, kept, mute)
	require.Len(t, events.Published(), 1)
	assert.True(t, proto.Equal(&chatv1.AccountMuted{
		AccountId: "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10",
		MutedAt:   timestamppb.New(at),
		Duration:  durationpb.New(90 * time.Minute),
	}, events.Published()[0]), "and never the network it holds")
}

func TestAMuteThatWasNotKeptPublishesNothing(t *testing.T) {
	events := cpbootstrap.NewRecordedEvents()
	refused := errors.New("postgres is down")

	_, err := publishing_mute.New(stubExecutor{err: refused}, events).Execute(t.Context(), mute_usecase.In{Account: bully})

	require.ErrorIs(t, err, refused)
	assert.Empty(t, events.Published())
}
