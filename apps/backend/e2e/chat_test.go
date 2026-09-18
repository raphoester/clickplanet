package e2e_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
)

// guestName is what the game calls an account with no username: the prefix and its 6 hex character code.
var guestName = regexp.MustCompile(`^guest_[0-9a-f]{6}$`)

// post sends a message, with the gamer's click token when it has one.
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

// react puts a reaction on a message, or takes it off, with the gamer's click token when it has one.
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
