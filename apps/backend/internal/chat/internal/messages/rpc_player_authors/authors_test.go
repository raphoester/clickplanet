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
)

// stubPlayer answers what the player module answers: the name it shows for an account.
type stubPlayer struct {
	playerv1connect.UnimplementedInternalServiceHandler

	names  map[string]string
	admins map[string]bool
	err    error
	asked  *string
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
		Name:  s.names[req.Msg.GetAccountId()],
		Admin: s.admins[req.Msg.GetAccountId()],
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

func TestItAnswersTheName(t *testing.T) {
	asked := new(string)
	player := stubPlayer{names: map[string]string{ada.String(): "Ada_L"}, admins: map[string]bool{ada.String(): true}, asked: asked}

	author, err := authors(t, player).Author(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, messages.Author{Name: "Ada_L", Admin: true}, author)
	assert.Equal(t, ada.String(), *asked)
}

func TestAPlayerModuleThatFailsIsAnError(t *testing.T) {
	player := stubPlayer{err: connect.NewError(connect.CodeInternal, errors.New("postgres is down")), asked: new(string)}

	_, err := authors(t, player).Author(t.Context(), ada)

	assert.ErrorContains(t, err, "failed to ask the player module")
}

func TestAnUnreachablePlayerModuleIsAnError(t *testing.T) {
	_, err := rpc_player_authors.New(dialer{err: errors.New("no internal listener")}).Author(t.Context(), ada)

	assert.ErrorContains(t, err, "failed to reach the player module")
}
