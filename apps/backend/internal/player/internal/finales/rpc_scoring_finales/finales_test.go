package rpc_scoring_finales_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales/rpc_scoring_finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type stubScoring struct {
	scoringv1connect.UnimplementedInternalServiceHandler

	table   *scoringv1.GetTableResponse
	players []*scoringv1.RoundPlayer
}

func (s stubScoring) GetTable(
	context.Context, *connect.Request[scoringv1.GetTableRequest],
) (*connect.Response[scoringv1.GetTableResponse], error) {
	return connect.NewResponse(s.table), nil
}

func (s stubScoring) GetRoundPlayers(
	context.Context, *connect.Request[scoringv1.GetRoundPlayersRequest],
) (*connect.Response[scoringv1.GetRoundPlayersResponse], error) {
	return connect.NewResponse(&scoringv1.GetRoundPlayersResponse{Players: s.players}), nil
}

func finalesOf(t *testing.T, scoring stubScoring) *rpc_scoring_finales.Finales {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(scoringv1connect.NewInternalServiceHandler(scoring))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_scoring_finales.New(scoringv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestAClosedTableIsEachFlagsPlace(t *testing.T) {
	scoring := stubScoring{table: &scoringv1.GetTableResponse{Closed: true, Table: []*scoringv1.TablePlace{
		{CountryId: "fr", Rank: 1, Points: 300}, {CountryId: "de", Rank: 2, Points: 250},
	}}}

	table, err := finalesOf(t, scoring).Table(t.Context(), "season-0")

	require.NoError(t, err)
	assert.Equal(t, []finales.Place{{Country: "fr", Rank: 1}, {Country: "de", Rank: 2}}, table)
}

func TestAnOpenTableIsAnError(t *testing.T) {
	scoring := stubScoring{table: &scoringv1.GetTableResponse{Table: []*scoringv1.TablePlace{{CountryId: "fr", Rank: 1}}}}

	_, err := finalesOf(t, scoring).Table(t.Context(), "season-0")

	assert.Error(t, err)
}

func TestThePlayersOfTheRoundAreItsFighters(t *testing.T) {
	ada := players.AccountID{15: 1}
	scoring := stubScoring{players: []*scoringv1.RoundPlayer{{AccountId: ada.String(), CountryId: "fr", Tiles: 3}}}

	fighters, err := finalesOf(t, scoring).Fighters(t.Context(), "season-0/finale")

	require.NoError(t, err)
	assert.Equal(t, []finales.Fighter{{Account: ada, Country: "fr"}}, fighters)
}

func TestAPlayerWithNoAccountIDIsAnError(t *testing.T) {
	scoring := stubScoring{players: []*scoringv1.RoundPlayer{{AccountId: "ada", CountryId: "fr", Tiles: 3}}}

	_, err := finalesOf(t, scoring).Fighters(t.Context(), "season-0/finale")

	assert.Error(t, err)
}
