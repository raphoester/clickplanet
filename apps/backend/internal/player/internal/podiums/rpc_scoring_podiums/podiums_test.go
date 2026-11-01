package rpc_scoring_podiums_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums/rpc_scoring_podiums"
)

type stubScoring struct {
	scoringv1connect.UnimplementedInternalServiceHandler

	asked     []string
	standings []*scoringv1.Standing
}

func (s *stubScoring) GetBoard(
	_ context.Context, req *connect.Request[scoringv1.GetBoardRequest],
) (*connect.Response[scoringv1.GetBoardResponse], error) {
	s.asked = append(s.asked, req.Msg.GetBookId()+"|"+req.Msg.GetCountryId())
	return connect.NewResponse(&scoringv1.GetBoardResponse{Standings: s.standings}), nil
}

func podiumsOf(t *testing.T, scoring *stubScoring) *rpc_scoring_podiums.Podiums {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(scoringv1connect.NewInternalServiceHandler(scoring))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_scoring_podiums.New(scoringv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestTheBoardOfEveryFlagIsEachPlayersRank(t *testing.T) {
	ada := players.AccountID{15: 1}
	bob := players.AccountID{15: 2}
	scoring := &stubScoring{standings: []*scoringv1.Standing{
		{Rank: 1, AccountId: ada.String(), Name: "Ada", Tiles: 900}, {Rank: 1, AccountId: bob.String(), Name: "Bob", Tiles: 900},
	}}

	entries, err := podiumsOf(t, scoring).Board(t.Context(), "season-0")

	require.NoError(t, err)
	assert.Equal(t, []podiums.Entry{{Account: ada, Rank: 1}, {Account: bob, Rank: 1}}, entries)
	assert.Equal(t, []string{"season-0|"}, scoring.asked, "the whole map's board, no flag's")
}

func TestAPlayerWithNoAccountIDIsAnError(t *testing.T) {
	scoring := &stubScoring{standings: []*scoringv1.Standing{{Rank: 1, AccountId: "ada"}}}

	_, err := podiumsOf(t, scoring).Board(t.Context(), "season-0")

	assert.Error(t, err)
}
