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
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_my_season_handler/my_season_query/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

type stubPlayer struct {
	playerv1connect.UnimplementedInternalServiceHandler

	known map[string]*playerv1.Author
	err   error
	asked *[]string
}

func (s stubPlayer) GetAuthors(
	_ context.Context,
	req *connect.Request[playerv1.GetAuthorsRequest],
) (*connect.Response[playerv1.GetAuthorsResponse], error) {
	*s.asked = append(*s.asked, req.Msg.GetAccountIds()...)
	if s.err != nil {
		return nil, s.err
	}
	found := make([]*playerv1.Author, 0, len(req.Msg.GetAccountIds()))
	for _, id := range req.Msg.GetAccountIds() {
		if author, known := s.known[id]; known {
			found = append(found, author)
		}
	}
	return connect.NewResponse(&playerv1.GetAuthorsResponse{Authors: found}), nil
}

var (
	ada     = standings.AccountID{15: 1}
	deleted = standings.AccountID{15: 2}
)

func authors(t *testing.T, player stubPlayer) *rpc_player_authors.Authors {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(playerv1connect.NewInternalServiceHandler(player))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_player_authors.New(playerv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestEachAccountTheModuleKnowsIsAnsweredAsItsAuthor(t *testing.T) {
	card := &playerv1.Author{
		AccountId: ada.String(), Name: "Ada_L", Admin: true, Color: playerv1.NameColor_NAME_COLOR_TEAL, Streak: 12,
		WornTitle: &playerv1.Title{Id: "og", Name: "OG"},
	}
	asked := new([]string)

	found, err := authors(t, stubPlayer{known: map[string]*playerv1.Author{ada.String(): card}, asked: asked}).
		Authors(t.Context(), []standings.AccountID{ada, deleted})

	require.NoError(t, err)
	require.Len(t, found, 1, "an account the player module cannot name is left out")
	assert.True(t, proto.Equal(card, found[ada]))
	assert.Equal(t, []string{ada.String(), deleted.String()}, *asked)
}

func TestNobodyToNameAsksNothing(t *testing.T) {
	asked := new([]string)

	found, err := authors(t, stubPlayer{asked: asked}).Authors(t.Context(), nil)

	require.NoError(t, err)
	assert.Empty(t, found)
	assert.Empty(t, *asked)
}

func TestAnAnswerForSomethingThatIsNotAnAccountIsAnError(t *testing.T) {
	player := stubPlayer{
		known: map[string]*playerv1.Author{ada.String(): {AccountId: "not-a-uuid", Name: "Ada_L"}},
		asked: new([]string),
	}

	_, err := authors(t, player).Authors(t.Context(), []standings.AccountID{ada})

	assert.ErrorContains(t, err, "which is not an account")
}

func TestAPlayerModuleThatFailsIsAnError(t *testing.T) {
	player := stubPlayer{err: connect.NewError(connect.CodeInternal, errors.New("postgres is down")), asked: new([]string)}

	_, err := authors(t, player).Authors(t.Context(), []standings.AccountID{ada})

	assert.ErrorContains(t, err, "failed to ask the player module")
}
