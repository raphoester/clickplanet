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

type stubPlayer struct {
	playerv1connect.UnimplementedInternalServiceHandler

	names  map[string]string
	admins map[string]bool
	colors map[string]playerv1.NameColor
	guests map[string]bool
	err    error
	asked  *string
}

func (s stubPlayer) IntroducePlayer(
	_ context.Context,
	req *connect.Request[playerv1.IntroducePlayerRequest],
) (*connect.Response[playerv1.IntroducePlayerResponse], error) {
	*s.asked = req.Msg.GetAccountId()
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&playerv1.IntroducePlayerResponse{Player: &playerv1.Player{
		AccountId: req.Msg.GetAccountId(),
		Name:      s.names[req.Msg.GetAccountId()],
		Admin:     s.admins[req.Msg.GetAccountId()],
		Color:     s.colors[req.Msg.GetAccountId()],
		Guest:     s.guests[req.Msg.GetAccountId()],
	}}), nil
}

func (s stubPlayer) GetPlayers(
	_ context.Context,
	req *connect.Request[playerv1.GetPlayersRequest],
) (*connect.Response[playerv1.GetPlayersResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	found := make([]*playerv1.Player, 0, len(req.Msg.GetAccountIds()))
	for _, id := range req.Msg.GetAccountIds() {
		found = append(found, &playerv1.Player{AccountId: id, Name: s.names[id], Color: s.colors[id]})
	}
	return connect.NewResponse(&playerv1.GetPlayersResponse{Players: found}), nil
}

var ada = messages.AccountID{15: 1}

func authors(t *testing.T, player stubPlayer) *rpc_player_authors.Authors {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(playerv1connect.NewInternalServiceHandler(player))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_player_authors.New(playerv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestItAnswersTheName(t *testing.T) {
	asked := new(string)
	player := stubPlayer{names: map[string]string{ada.String(): "Ada_L"}, admins: map[string]bool{ada.String(): true}, asked: asked}

	author, err := authors(t, player).Author(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, messages.AuthorOf("Ada_L", true, 0), author)
	assert.Equal(t, ada.String(), *asked)
}

func TestItAnswersTheColor(t *testing.T) {
	player := stubPlayer{
		names:  map[string]string{ada.String(): "Ada_L"},
		colors: map[string]playerv1.NameColor{ada.String(): playerv1.NameColor_NAME_COLOR_TEAL},
		asked:  new(string),
	}

	author, err := authors(t, player).Author(t.Context(), ada)
	require.NoError(t, err)
	many, err := authors(t, player).Authors(t.Context(), []messages.AccountID{ada})
	require.NoError(t, err)

	assert.Equal(t, messages.AuthorOf("Ada_L", false, int32(playerv1.NameColor_NAME_COLOR_TEAL)), author)
	assert.Equal(t, author, many[ada])
}

func TestAPlayerModuleThatFailsIsAnError(t *testing.T) {
	player := stubPlayer{err: connect.NewError(connect.CodeInternal, errors.New("postgres is down")), asked: new(string)}

	_, err := authors(t, player).Author(t.Context(), ada)

	assert.ErrorContains(t, err, "failed to ask the player module")
}

func TestAGuestIsAnsweredAsOne(t *testing.T) {
	player := stubPlayer{
		names: map[string]string{ada.String(): "guest_0b1c2d"}, guests: map[string]bool{ada.String(): true}, asked: new(string),
	}

	author, err := authors(t, player).Author(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, messages.GuestOf("guest_0b1c2d"), author)
}
