package rpc_players_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_round_handler/round_query/rpc_players"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
)

type stubPlayer struct {
	playerv1connect.UnimplementedInternalServiceHandler

	known map[string]*playerv1.Player
	err   error
	asked *[]string
}

func (s stubPlayer) GetPlayers(
	_ context.Context,
	req *connect.Request[playerv1.GetPlayersRequest],
) (*connect.Response[playerv1.GetPlayersResponse], error) {
	*s.asked = append(*s.asked, req.Msg.GetAccountIds()...)
	if s.err != nil {
		return nil, s.err
	}
	found := make([]*playerv1.Player, 0, len(req.Msg.GetAccountIds()))
	for _, id := range req.Msg.GetAccountIds() {
		if player, known := s.known[id]; known {
			found = append(found, player)
		}
	}
	return connect.NewResponse(&playerv1.GetPlayersResponse{Players: found}), nil
}

var (
	ada     = standings.AccountID{15: 1}
	deleted = standings.AccountID{15: 2}
)

func players(t *testing.T, player stubPlayer) *rpc_players.Players {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(playerv1connect.NewInternalServiceHandler(player))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_players.New(playerv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestEachAccountTheModuleKnowsIsAnsweredAsItsPlayer(t *testing.T) {
	card := &playerv1.Player{
		AccountId: ada.String(), Name: "Ada_L", Admin: true, Color: playerv1.NameColor_NAME_COLOR_TEAL,
	}
	asked := new([]string)

	found, err := players(t, stubPlayer{known: map[string]*playerv1.Player{ada.String(): card}, asked: asked}).
		Players(t.Context(), []standings.AccountID{ada, deleted})

	require.NoError(t, err)
	require.Len(t, found, 1, "an account the player module cannot name is left out")
	assert.True(t, proto.Equal(card, found[ada]))
	assert.Equal(t, []string{ada.String(), deleted.String()}, *asked)
}

func TestNobodyToNameAsksNothing(t *testing.T) {
	asked := new([]string)

	found, err := players(t, stubPlayer{asked: asked}).Players(t.Context(), nil)

	require.NoError(t, err)
	assert.Empty(t, found)
	assert.Empty(t, *asked)
}

func TestAnAnswerForSomethingThatIsNotAnAccountIsAnError(t *testing.T) {
	player := stubPlayer{
		known: map[string]*playerv1.Player{ada.String(): {AccountId: "not-a-uuid", Name: "Ada_L"}},
		asked: new([]string),
	}

	_, err := players(t, player).Players(t.Context(), []standings.AccountID{ada})

	assert.ErrorContains(t, err, "which is not an account")
}

func TestAPlayerModuleThatFailsIsAnError(t *testing.T) {
	player := stubPlayer{err: connect.NewError(connect.CodeInternal, errors.New("postgres is down")), asked: new([]string)}

	_, err := players(t, player).Players(t.Context(), []standings.AccountID{ada})

	assert.ErrorContains(t, err, "failed to ask the player module")
}
