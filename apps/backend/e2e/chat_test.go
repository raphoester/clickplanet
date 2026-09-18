package e2e_test

import (
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
)

// post sends a message as the name typed, with the gamer's click token when it has one.
func (p *gamer) post(typed string) (*chatv1.ChatMessage, error) {
	p.t.Helper()

	req := connect.NewRequest(&chatv1.SendMessageRequest{AuthorName: typed, AuthorId: "browser-1", CountryId: "fr", Text: "hello"})
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

	message, err := ada.post("Bob")

	require.NoError(t, err)
	assert.Equal(t, "Ada_L", message.GetAuthorName(), "the typed name is not read")
	assert.NotEmpty(t, message.GetAuthorTag())
}

func TestAPlayerWithNoUsernamePostsAsAGuest(t *testing.T) {
	game := startGame(t)

	message, err := game.newPlayer(t).post("Bob")

	require.NoError(t, err)
	assert.Equal(t, "guest_Bob", message.GetAuthorName())
}

func TestASenderWithNoTokenPostsAsAGuest(t *testing.T) {
	game := startGame(t)
	nobody := &gamer{t: t, stack: game}

	message, err := nobody.post("Bob")

	require.NoError(t, err)
	assert.Equal(t, "guest_Bob", message.GetAuthorName())
	assert.Len(t, message.GetAuthorTag(), 6, "the player module tags a sender with no account too")
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

func TestAPlayerAndAGuestOnOneAddressAreTwoReactors(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")
	_, err := ada.setName("Ada_L")
	require.NoError(t, err)
	nobody := &gamer{t: t, stack: game}
	message, err := nobody.post("Bob")
	require.NoError(t, err)

	counts, err := ada.react(message.GetId(), chatv1.Reaction_REACTION_CLOWN, true)
	require.NoError(t, err)
	assert.True(t, counts[0].GetMine())

	counts, err = nobody.react(message.GetId(), chatv1.Reaction_REACTION_CLOWN, true)
	require.NoError(t, err)
	require.Len(t, counts, 1)
	assert.Equal(t, uint32(2), counts[0].GetCount(), "the account is one reactor, the address another")

	counts, err = ada.react(message.GetId(), chatv1.Reaction_REACTION_CLOWN, false)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), counts[0].GetCount())
	assert.False(t, counts[0].GetMine())

	reactions := nobody.history()[0].GetReactions()
	require.Len(t, reactions, 1)
	assert.Equal(t, chatv1.Reaction_REACTION_CLOWN, reactions[0].GetReaction())
	assert.True(t, reactions[0].GetMine(), "the history says which reactions are the caller's")
}

func TestAReactionToNoMessageIsNotFound(t *testing.T) {
	game := startGame(t)

	_, err := game.newPlayer(t).react("no-such-message", chatv1.Reaction_REACTION_SKULL, true)

	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
