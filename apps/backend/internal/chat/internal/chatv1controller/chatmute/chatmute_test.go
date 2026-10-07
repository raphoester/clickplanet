package chatmute_test

import (
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatmute"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
)

func TestAMutedCallerIsDeniedAndToldUntilWhen(t *testing.T) {
	at := time.Date(2026, 10, 7, 20, 0, 0, 0, time.UTC)
	refusal := mutes.NewMute(mutes.MuteID{1}, mutes.NewCaller(messages.AccountID{15: 1}, ""), at, time.Hour).Refusal()

	refused, ok := chatmute.Refusal(fmt.Errorf("wrapped: %w", refusal))
	require.True(t, ok)

	assert.Equal(t, connect.CodePermissionDenied, refused.Code())
	assert.NotContains(t, refused.Message(), "2026", "the time travels in the detail")
	require.Len(t, refused.Details(), 1)
	detail, err := refused.Details()[0].Value()
	require.NoError(t, err)
	mute, isMute := detail.(*chatv1.MuteRefusal)
	require.True(t, isMute)
	assert.Equal(t, at.Add(time.Hour).UnixMilli(), mute.GetMutedUntilUnixMs())
}

func TestAnyOtherErrorIsNoMuteRefusal(t *testing.T) {
	_, ok := chatmute.Refusal(assert.AnError)

	assert.False(t, ok)
}
