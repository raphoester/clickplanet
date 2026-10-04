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
	streak map[string]uint32
	titles map[string]*playerv1.Title
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
		Name:      s.names[req.Msg.GetAccountId()],
		Admin:     s.admins[req.Msg.GetAccountId()],
		Color:     s.colors[req.Msg.GetAccountId()],
		Streak:    s.streak[req.Msg.GetAccountId()],
		WornTitle: s.titles[req.Msg.GetAccountId()],
	}), nil
}

func (s stubPlayer) GetAuthors(
	_ context.Context,
	req *connect.Request[playerv1.GetAuthorsRequest],
) (*connect.Response[playerv1.GetAuthorsResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	found := make([]*playerv1.Author, 0, len(req.Msg.GetAccountIds()))
	for _, id := range req.Msg.GetAccountIds() {
		found = append(found, &playerv1.Author{AccountId: id, Name: s.names[id], WornTitle: s.titles[id]})
	}
	return connect.NewResponse(&playerv1.GetAuthorsResponse{Authors: found}), nil
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
	assert.Equal(t, messages.AuthorOf("Ada_L", true, 0, 0, messages.Title{}), author)
	assert.Equal(t, ada.String(), *asked)
}

func TestItAnswersTheColorAndTheStreak(t *testing.T) {
	player := stubPlayer{
		names:  map[string]string{ada.String(): "Ada_L"},
		colors: map[string]playerv1.NameColor{ada.String(): playerv1.NameColor_NAME_COLOR_TEAL},
		streak: map[string]uint32{ada.String(): 12},
		asked:  new(string),
	}

	author, err := authors(t, player).Author(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, messages.AuthorOf("Ada_L", false, int32(playerv1.NameColor_NAME_COLOR_TEAL), 12, messages.Title{}), author)
}

var settler = &playerv1.Title{
	Id: "settler", Name: "Settler", Rank: &playerv1.Rank{TrackId: "conquest", TrackName: "Conquest", Number: 1, Count: 5},
}

func TestItAnswersTheTitleWorn(t *testing.T) {
	player := stubPlayer{
		names:  map[string]string{ada.String(): "Ada_L"},
		titles: map[string]*playerv1.Title{ada.String(): settler},
		asked:  new(string),
	}

	author, err := authors(t, player).Author(t.Context(), ada)
	require.NoError(t, err)
	many, err := authors(t, player).Authors(t.Context(), []messages.AccountID{ada})
	require.NoError(t, err)

	want := messages.TitleOf("settler", "Settler", messages.RankOf("conquest", "Conquest", 1, 5))
	assert.Equal(t, want, author.Title())
	assert.Equal(t, want, many[ada].Title())
}

func TestAnAuthorWearingNothingHasNoTitle(t *testing.T) {
	player := stubPlayer{names: map[string]string{ada.String(): "Ada_L"}, asked: new(string)}

	author, err := authors(t, player).Author(t.Context(), ada)

	require.NoError(t, err)
	assert.True(t, author.Title().Empty())
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
