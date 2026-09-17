package rpc_player_usernames_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/rpc_player_usernames"
)

// stubPlayer answers what the player module answers: the names it holds, leaving out the accounts with none.
type stubPlayer struct {
	playerv1connect.UnimplementedInternalServiceHandler

	names map[string]string
	err   error
}

func (s stubPlayer) GetNames(
	_ context.Context,
	req *connect.Request[playerv1.GetNamesRequest],
) (*connect.Response[playerv1.GetNamesResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	names := map[string]string{}
	for _, id := range req.Msg.GetAccountIds() {
		if name, ok := s.names[id]; ok {
			names[id] = name
		}
	}
	return connect.NewResponse(&playerv1.GetNamesResponse{Names: names}), nil
}

type dialer struct {
	client connect.HTTPClient
	url    string
	err    error
}

func (d dialer) Dial() (connect.HTTPClient, string, error) {
	return d.client, d.url, d.err
}

var (
	ada   = messages.AccountID{15: 1}
	guest = messages.AccountID{15: 2}
)

func usernames(t *testing.T, player stubPlayer) *rpc_player_usernames.Usernames {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(playerv1connect.NewInternalServiceHandler(player))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_player_usernames.New(dialer{client: server.Client(), url: server.URL})
}

func TestItAnswersTheAccountsUsername(t *testing.T) {
	username, found, err := usernames(t, stubPlayer{names: map[string]string{ada.String(): "Ada_L"}}).Username(t.Context(), ada)

	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "Ada_L", username)
}

func TestAnAccountWithNoUsernameIsNotFound(t *testing.T) {
	_, found, err := usernames(t, stubPlayer{names: map[string]string{ada.String(): "Ada_L"}}).Username(t.Context(), guest)

	require.NoError(t, err)
	assert.False(t, found)
}

func TestAPlayerModuleThatFailsIsAnError(t *testing.T) {
	_, _, err := usernames(t, stubPlayer{err: connect.NewError(connect.CodeNotFound, errors.New("player is off"))}).
		Username(t.Context(), ada)

	assert.ErrorContains(t, err, "failed to ask the player module")
}

func TestAnUnreachablePlayerModuleIsAnError(t *testing.T) {
	_, _, err := rpc_player_usernames.New(dialer{err: errors.New("no internal listener")}).Username(t.Context(), ada)

	assert.ErrorContains(t, err, "failed to reach the player module")
}
