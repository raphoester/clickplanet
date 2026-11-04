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

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_reports_handler/reports_query/rpc_players"
)

type stubPlayer struct {
	playerv1connect.UnimplementedInternalServiceHandler

	names map[string]string
	err   error
	asks  *int
}

func (s stubPlayer) GetPlayers(
	_ context.Context,
	req *connect.Request[playerv1.GetPlayersRequest],
) (*connect.Response[playerv1.GetPlayersResponse], error) {
	*s.asks++
	if s.err != nil {
		return nil, s.err
	}
	found := make([]*playerv1.Player, 0, len(req.Msg.GetAccountIds()))
	for _, id := range req.Msg.GetAccountIds() {
		if name, named := s.names[id]; named {
			found = append(found, &playerv1.Player{AccountId: id, Name: name})
		}
	}
	return connect.NewResponse(&playerv1.GetPlayersResponse{Players: found}), nil
}

var (
	ada  = cpsession.AccountID{15: 1}
	gone = cpsession.AccountID{15: 2}
)

func players(t *testing.T, player stubPlayer) *rpc_players.Players {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(playerv1connect.NewInternalServiceHandler(player))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_players.New(playerv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestTheAccountsThePlayerModuleCanNameAreAnsweredAndTheOthersLeftOut(t *testing.T) {
	asks := 0

	found, err := players(t, stubPlayer{names: map[string]string{ada.String(): "Ada_L"}, asks: &asks}).
		Players(t.Context(), []cpsession.AccountID{ada, gone})

	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, "Ada_L", found[ada].GetName())
	assert.Equal(t, 1, asks)
}

func TestNobodyToNameAsksNothing(t *testing.T) {
	asks := 0

	found, err := players(t, stubPlayer{asks: &asks}).Players(t.Context(), nil)

	require.NoError(t, err)
	assert.Empty(t, found)
	assert.Zero(t, asks)
}

func TestAFailureToAskIsAnError(t *testing.T) {
	asks := 0

	_, err := players(t, stubPlayer{err: connect.NewError(connect.CodeInternal, errors.New("postgres is down")), asks: &asks}).
		Players(t.Context(), []cpsession.AccountID{ada})

	require.Error(t, err)
}
