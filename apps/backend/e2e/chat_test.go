package e2e_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
)

var guestName = regexp.MustCompile(`^guest_[0-9a-f]{6}$`)

func (p *gamer) post() (*chatv1.ChatMessage, error) {
	p.t.Helper()

	req := connect.NewRequest(&chatv1.SendMessageRequest{AuthorId: "browser-1", CountryId: "fr", Text: "hello"})
	p.send(req.Header())
	res, err := chatv1connect.NewChatServiceClient(http.DefaultClient, p.stack.baseURL).SendMessage(p.t.Context(), req)
	if err != nil {
		return nil, fmt.Errorf("SendMessage failed: %w", err)
	}
	return res.Msg.GetMessage(), nil
}

func TestAPlayerWithAUsernamePostsUnderIt(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")
	_, err := ada.setName("Ada_L")
	require.NoError(t, err)

	message, err := ada.post()

	require.NoError(t, err)
	assert.Equal(t, "Ada_L", message.GetAuthorName())
}

func TestAPlayerPostsInTheColorItChoseAndWithItsStreak(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")
	_, err := ada.setName("Ada_L")
	require.NoError(t, err)
	require.NoError(t, ada.setColor(playerv1.NameColor_NAME_COLOR_TEAL))
	ada.click(1, "fr")
	require.Eventually(t, func() bool { return ada.stats().GetStreakCurrent() == 1 }, 5*time.Second, 20*time.Millisecond)

	message, err := ada.post()

	require.NoError(t, err)
	assert.Equal(t, playerv1.NameColor_NAME_COLOR_TEAL, message.GetAuthorColor())
	assert.Equal(t, uint32(1), message.GetAuthorStreak())
}

func TestEachMessageSentClimbsTheChatterTrackAndTakesNoTile(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")

	for range 3 {
		_, err := ada.post()
		require.NoError(t, err)
	}

	chatter := func() *playerv1.Track {
		req := connect.NewRequest(&playerv1.GetTitlesRequest{})
		ada.send(req.Header())
		res, err := ada.players().GetTitles(t.Context(), req)
		require.NoError(t, err)
		for _, track := range res.Msg.GetTracks() {
			if track.GetId() == "chatter" {
				return track
			}
		}
		return nil
	}
	require.Eventually(t, func() bool { return chatter().GetProgress() == 3 }, 5*time.Second, 20*time.Millisecond,
		"chat publishes each message and player counts it")
	assert.Zero(t, ada.stats().GetTilesTaken())
	assert.Zero(t, ada.stats().GetStreakCurrent(), "a message is no take")
}

func TestAGuestMayNotChooseAColor(t *testing.T) {
	game := startGame(t)

	err := game.newPlayer(t).setColor(playerv1.NameColor_NAME_COLOR_TEAL)

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestAPlayerWithNoUsernamePostsUnderItsGuestCodeEveryTime(t *testing.T) {
	game := startGame(t)
	bob := game.newPlayer(t)

	first, err := bob.post()
	require.NoError(t, err)
	second, err := bob.post()
	require.NoError(t, err)

	assert.Regexp(t, guestName, first.GetAuthorName())
	assert.Equal(t, first.GetAuthorName(), second.GetAuthorName(), "the code is drawn once and kept")
	other, err := game.newPlayer(t).post()
	require.NoError(t, err)
	assert.NotEqual(t, first.GetAuthorName(), other.GetAuthorName(), "each account has its own code")
}

func TestASenderWithNoTokenIsUnauthenticated(t *testing.T) {
	game := startGame(t)
	nobody := &gamer{t: t, stack: game}

	_, err := nobody.post()

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func (p *gamer) react(messageID string, reaction chatv1.Reaction, on bool) ([]*chatv1.ReactionCount, error) {
	p.t.Helper()

	req := connect.NewRequest(&chatv1.ReactRequest{MessageId: messageID, Reaction: reaction, On: on})
	p.send(req.Header())
	res, err := chatv1connect.NewChatServiceClient(http.DefaultClient, p.stack.baseURL).React(p.t.Context(), req)
	if err != nil {
		return nil, fmt.Errorf("React failed: %w", err)
	}
	return res.Msg.GetReactions(), nil
}

func (p *gamer) history() []*chatv1.ChatMessage {
	p.t.Helper()

	req := connect.NewRequest(&chatv1.GetHistoryRequest{})
	p.send(req.Header())
	res, err := chatv1connect.NewChatServiceClient(http.DefaultClient, p.stack.baseURL).GetHistory(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetMessages()
}

func TestEachAccountIsItsOwnReactorAndNoAccountIsRefused(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")
	_, err := ada.setName("Ada_L")
	require.NoError(t, err)
	bob := game.newPlayer(t)
	message, err := bob.post()
	require.NoError(t, err)

	counts, err := ada.react(message.GetId(), chatv1.Reaction_REACTION_CLOWN, true)
	require.NoError(t, err)
	assert.True(t, counts[0].GetMine())

	counts, err = bob.react(message.GetId(), chatv1.Reaction_REACTION_CLOWN, true)
	require.NoError(t, err)
	require.Len(t, counts, 1)
	assert.Equal(t, uint32(2), counts[0].GetCount(), "two accounts on one address are two reactors")

	counts, err = ada.react(message.GetId(), chatv1.Reaction_REACTION_CLOWN, false)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), counts[0].GetCount())
	assert.False(t, counts[0].GetMine())

	reactions := bob.history()[0].GetReactions()
	require.Len(t, reactions, 1)
	assert.Equal(t, chatv1.Reaction_REACTION_CLOWN, reactions[0].GetReaction())
	assert.True(t, reactions[0].GetMine(), "the history says which reactions are the caller's")

	nobody := &gamer{t: t, stack: game}
	_, err = nobody.react(message.GetId(), chatv1.Reaction_REACTION_CLOWN, true)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestAReactionToNoMessageIsNotFound(t *testing.T) {
	game := startGame(t)

	_, err := game.newPlayer(t).react("no-such-message", chatv1.Reaction_REACTION_SKULL, true)

	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
