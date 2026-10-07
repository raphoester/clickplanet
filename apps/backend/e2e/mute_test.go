package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
)

func (s gameStack) mute(t *testing.T, account string) *chatv1.MuteResponse {
	t.Helper()

	res, err := chatv1connect.NewAdminServiceClient(http.DefaultClient, s.adminURL).Mute(t.Context(),
		connect.NewRequest(&chatv1.MuteRequest{AccountId: account}))
	require.NoError(t, err)
	return res.Msg
}

func (p *gamer) announcements() []*chatv1.Announcement {
	p.t.Helper()

	req := connect.NewRequest(&chatv1.GetHistoryRequest{})
	p.send(req.Header())
	res, err := chatv1connect.NewChatServiceClient(http.DefaultClient, p.stack.baseURL).GetHistory(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetAnnouncements()
}

func mutedUntil(t *testing.T, err error) time.Time {
	t.Helper()

	var denied *connect.Error
	require.ErrorAs(t, err, &denied)
	require.Equal(t, connect.CodePermissionDenied, denied.Code())
	require.Len(t, denied.Details(), 1)
	detail, err := denied.Details()[0].Value()
	require.NoError(t, err)
	refusal, ok := detail.(*chatv1.MuteRefusal)
	require.True(t, ok)
	return time.UnixMilli(refusal.GetMutedUntilUnixMs())
}

func TestAMutedPlayerCanNeitherPostNorReactNorDodgeItWithAFreshGuestAndEveryoneIsTold(t *testing.T) {
	game := startGame(t)
	newcomer, bully := game.newPlayer(t), game.newPlayer(t)
	welcome, err := newcomer.post()
	require.NoError(t, err)
	_, err = bully.post()
	require.NoError(t, err)

	muted := game.mute(t, bully.account())

	assert.Regexp(t, guestName, muted.GetName())
	assert.Equal(t, callerIP, muted.GetScope(), "the network of the latest message")
	assert.WithinDuration(t, time.Now().Add(time.Hour), muted.GetMutedUntil().AsTime(), time.Minute)

	_, err = bully.post()
	assert.Equal(t, muted.GetMutedUntil().AsTime().UnixMilli(), mutedUntil(t, err).UnixMilli())
	_, err = bully.react(welcome.GetId(), chatv1.Reaction_REACTION_CLOWN, true)
	assert.Equal(t, muted.GetMutedUntil().AsTime().UnixMilli(), mutedUntil(t, err).UnixMilli())
	_, err = game.newPlayer(t).post()
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "a fresh guest on the muted network")

	announcements := newcomer.announcements()
	require.Len(t, announcements, 1)
	assert.Equal(t, "mute", announcements[0].GetKind())
	assert.JSONEq(t, `{"name":"`+muted.GetName()+`","seconds":3600}`, announcements[0].GetPayload())
}

func TestAPlayerThatNeverPostedIsMutedAloneAndTheOthersStillChat(t *testing.T) {
	game := startGame(t)
	quiet, other := game.newPlayer(t), game.newPlayer(t)

	muted := game.mute(t, quiet.account())

	assert.Empty(t, muted.GetScope(), "no message, no network")
	_, err := quiet.post()
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	_, err = other.post()
	assert.NoError(t, err)
}
