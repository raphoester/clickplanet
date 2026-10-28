package rpc_scoring_results_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/rpc_scoring_results"
)

type stubScoring struct {
	scoringv1connect.UnimplementedInternalServiceHandler

	rounds map[string]*scoringv1.Round
	tables map[string]*scoringv1.GetTableResponse
}

func (s stubScoring) GetRound(
	_ context.Context,
	req *connect.Request[scoringv1.GetRoundRequest],
) (*connect.Response[scoringv1.GetRoundResponse], error) {
	round, ok := s.rounds[req.Msg.GetRoundId()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no such round"))
	}
	return connect.NewResponse(&scoringv1.GetRoundResponse{Round: round}), nil
}

func (s stubScoring) GetTable(
	_ context.Context,
	req *connect.Request[scoringv1.GetTableRequest],
) (*connect.Response[scoringv1.GetTableResponse], error) {
	table, ok := s.tables[req.Msg.GetBookId()]
	if !ok {
		return connect.NewResponse(&scoringv1.GetTableResponse{}), nil
	}
	return connect.NewResponse(table), nil
}

func results(t *testing.T, scoring stubScoring) *rpc_scoring_results.Results {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(scoringv1connect.NewInternalServiceHandler(scoring))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_scoring_results.New(scoringv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestARoundsResultsAreItsStandings(t *testing.T) {
	scoring := stubScoring{rounds: map[string]*scoringv1.Round{"season-0/day-5": {Standings: []*scoringv1.RoundStanding{
		{Rank: 1, CountryId: "fr", Share: 0.4, Points: 25},
		{Rank: 2, CountryId: "de", Share: 0.3, Points: 18},
	}}}}

	places, err := results(t, scoring).Round(t.Context(), "season-0/day-5")

	require.NoError(t, err)
	assert.Equal(t, []announcements.Place{announcements.PlaceOf("fr", 1, 25), announcements.PlaceOf("de", 2, 18)}, places)
}

func TestARoundScoringDoesNotKnowIsAnError(t *testing.T) {
	_, err := results(t, stubScoring{}).Round(t.Context(), "season-0/day-5")

	assert.ErrorContains(t, err, "failed to ask scoring for a round")
}

func TestAClosedBooksTableIsItsPlaces(t *testing.T) {
	scoring := stubScoring{tables: map[string]*scoringv1.GetTableResponse{"season-0": {Closed: true, Table: []*scoringv1.TablePlace{
		{CountryId: "de", Rank: 1, Points: 118, RoundsWon: 2, TiebreakPoints: 75},
		{CountryId: "fr", Rank: 2, Points: 97},
	}}}}

	places, err := results(t, scoring).Table(t.Context(), "season-0")

	require.NoError(t, err)
	assert.Equal(t, []announcements.Place{announcements.PlaceOf("de", 1, 118), announcements.PlaceOf("fr", 2, 97)}, places)
}

func TestAnOpenBooksTableIsRefused(t *testing.T) {
	scoring := stubScoring{tables: map[string]*scoringv1.GetTableResponse{"season-0": {Table: []*scoringv1.TablePlace{
		{CountryId: "de", Rank: 1, Points: 118},
	}}}}

	_, err := results(t, scoring).Table(t.Context(), "season-0")

	assert.ErrorContains(t, err, "not closed")
}
