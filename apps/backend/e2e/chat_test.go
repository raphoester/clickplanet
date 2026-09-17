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
}
