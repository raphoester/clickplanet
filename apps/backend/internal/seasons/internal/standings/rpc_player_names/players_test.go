package rpc_player_names_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/rpc_player_names"
)

type stubPlayer struct {
	playerv1connect.UnimplementedInternalServiceHandler

	authors []*playerv1.Author
	err     error
	asked   *[]string
}

func (s stubPlayer) GetAuthors(
	_ context.Context,
	req *connect.Request[playerv1.GetAuthorsRequest],
) (*connect.Response[playerv1.GetAuthorsResponse], error) {
	*s.asked = append(*s.asked, req.Msg.GetAccountIds()...)
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&playerv1.GetAuthorsResponse{Authors: s.authors}), nil
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
	ada   = standings.AccountID{0: 1, 15: 1}
	guest = standings.AccountID{0: 1, 15: 2}
)

func playersOver(t *testing.T, player stubPlayer) *rpc_player_names.Players {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(playerv1connect.NewInternalServiceHandler(player))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_player_names.New(dialer{client: server.Client(), url: server.URL})
}

func TestEachAccountIsAnsweredWithItsNameColorAndWhetherItIsAGuest(t *testing.T) {
	asked := new([]string)
	players := playersOver(t, stubPlayer{asked: asked, authors: []*playerv1.Author{
		{AccountId: ada.String(), Name: "Ada", Color: playerv1.NameColor_NAME_COLOR_PINK},
		{AccountId: guest.String(), Name: "guest_91aa3d", Guest: true},
	}})

	found, err := players.Players(t.Context(), []standings.AccountID{ada, guest})

	require.NoError(t, err)
	assert.Equal(t, map[standings.AccountID]standings.Player{
		ada:   {Name: "Ada", Color: standings.Color(playerv1.NameColor_NAME_COLOR_PINK)},
		guest: {Name: "guest_91aa3d", Guest: true},
	}, found)
	assert.Equal(t, []string{ada.String(), guest.String()}, *asked)
}

func TestNobodyAskedIsNoCall(t *testing.T) {
	asked := new([]string)

	found, err := playersOver(t, stubPlayer{asked: asked}).Players(t.Context(), []standings.AccountID{})

	require.NoError(t, err)
	assert.Empty(t, found)
	assert.Empty(t, *asked)
}

func TestAFailureToAskIsAnError(t *testing.T) {
	refused := connect.NewError(connect.CodeUnavailable, errors.New("down"))

	_, err := playersOver(t, stubPlayer{asked: new([]string), err: refused}).Players(t.Context(), []standings.AccountID{ada})

	assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

func TestNoInternalListenerIsAnError(t *testing.T) {
	refused := errors.New("no internal listener")

	_, err := rpc_player_names.New(dialer{err: refused}).Players(t.Context(), []standings.AccountID{ada})

	assert.ErrorIs(t, err, refused)
}

func TestAnAnswerForNoAccountIsAnError(t *testing.T) {
	players := playersOver(t, stubPlayer{asked: new([]string), authors: []*playerv1.Author{{AccountId: "nobody", Name: "Ada"}}})

	_, err := players.Players(t.Context(), []standings.AccountID{ada})

	assert.ErrorIs(t, err, standings.ErrInvalidAccount)
}
