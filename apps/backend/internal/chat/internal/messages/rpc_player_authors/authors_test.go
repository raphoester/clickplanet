package rpc_player_authors_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// stubPlayer answers what the player module answers: the name an account holds, and a tag for any address.
type stubPlayer struct {
	playerv1connect.UnimplementedInternalServiceHandler

	names map[string]string
	err   error
	asked *string
}

func (s stubPlayer) GetAuthor(
	_ context.Context,
	req *connect.Request[playerv1.GetAuthorRequest],
) (*connect.Response[playerv1.GetAuthorResponse], error) {
	*s.asked = req.Msg.GetAccountId()
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&playerv1.GetAuthorResponse{
		Username: s.names[req.Msg.GetAccountId()],
		Tag:      "tag-of-" + req.Msg.GetIp(),
	}), nil
}

type dialer struct {
	client connect.HTTPClient
	url    string
	err    error
}

func (d dialer) Dial() (connect.HTTPClient, string, error) {
	return d.client, d.url, d.err
}

var ada = messages.AccountID{15: 1}

func authors(t *testing.T, player stubPlayer) *rpc_player_authors.Authors {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(playerv1connect.NewInternalServiceHandler(player))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_player_authors.New(dialer{client: server.Client(), url: server.URL})
}

func TestItAnswersTheUsernameAndTheTag(t *testing.T) {
	asked := new(string)
	player := stubPlayer{names: map[string]string{ada.String(): "Ada_L"}, asked: asked}

	author, err := authors(t, player).Author(t.Context(), ada, "1.2.3.4")

	require.NoError(t, err)
	assert.Equal(t, messages.Author{Username: "Ada_L", Tag: "tag-of-1.2.3.4"}, author)
	assert.Equal(t, ada.String(), *asked)
}

func TestNoAccountIsAskedWithAnEmptyId(t *testing.T) {
	asked := new(string)

	author, err := authors(t, stubPlayer{asked: asked}).Author(t.Context(), cpsession.NoAccount, "1.2.3.4")

	require.NoError(t, err)
	assert.Equal(t, messages.Author{Tag: "tag-of-1.2.3.4"}, author)
	assert.Empty(t, *asked)
}

func TestAPlayerModuleThatFailsIsAnError(t *testing.T) {
	player := stubPlayer{err: connect.NewError(connect.CodeInternal, errors.New("postgres is down")), asked: new(string)}

	_, err := authors(t, player).Author(t.Context(), ada, "1.2.3.4")

	assert.ErrorContains(t, err, "failed to ask the player module")
}

func TestAnUnreachablePlayerModuleIsAnError(t *testing.T) {
	_, err := rpc_player_authors.New(dialer{err: errors.New("no internal listener")}).Author(t.Context(), ada, "1.2.3.4")

	assert.ErrorContains(t, err, "failed to reach the player module")
}
