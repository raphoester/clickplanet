package rpc_scoring_rounds_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings/rpc_scoring_rounds"
)

type stubScoring struct {
	scoringv1connect.UnimplementedInternalServiceHandler

	rounds map[string]*scoringv1.Round
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

func rounds(t *testing.T, scoring stubScoring) *rpc_scoring_rounds.Rounds {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(scoringv1connect.NewInternalServiceHandler(scoring))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_scoring_rounds.New(scoringv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestAScoredDayIsItsResultsAndTheTableAfterIt(t *testing.T) {
	scoring := stubScoring{rounds: map[string]*scoringv1.Round{"season-1/day-3": {
		EndedAtUnixMs: 1793570400000,
		Standings: []*scoringv1.RoundStanding{
			{Rank: 1, CountryId: "fr", Share: 0.4, Points: 25},
			{Rank: 2, CountryId: "de", Share: 0.3, Points: 18},
		},
		Before: []*scoringv1.Score{{Rank: 1, CountryId: "de", Points: 50}},
		After: []*scoringv1.Score{
			{Rank: 1, CountryId: "de", Points: 68, RoundsWon: 2},
			{Rank: 2, CountryId: "fr", Points: 25, RoundsWon: 1},
		},
	}}}

	ranking, err := rounds(t, scoring).RankingAfter(t.Context(), "season-1/day-3")

	require.NoError(t, err)
	assert.Equal(t, pings.Ranking{
		Won:   []pings.Place{{Flag: "fr", Rank: 1, Points: 25}, {Flag: "de", Rank: 2, Points: 18}},
		Table: []pings.Place{{Flag: "de", Rank: 1, Points: 68}, {Flag: "fr", Rank: 2, Points: 25}},
	}, ranking)
}

func TestARoundThatRunsIsNotScored(t *testing.T) {
	scoring := stubScoring{rounds: map[string]*scoringv1.Round{"season-1/day-3": {
		Standings: []*scoringv1.RoundStanding{{Rank: 1, CountryId: "fr", Share: 0.4, Points: 25}},
	}}}

	_, err := rounds(t, scoring).RankingAfter(t.Context(), "season-1/day-3")

	assert.ErrorIs(t, err, rpc_scoring_rounds.ErrNotScored)
}

func TestARoundScoringDoesNotKnowOrAFlagNobodyHasIsAnError(t *testing.T) {
	scoring := stubScoring{rounds: map[string]*scoringv1.Round{"season-1/day-3": {
		EndedAtUnixMs: 1793570400000,
		Standings:     []*scoringv1.RoundStanding{{Rank: 1, CountryId: "zz", Points: 25}},
	}}}

	_, err := rounds(t, scoring).RankingAfter(t.Context(), "season-1/day-4")
	require.ErrorContains(t, err, "failed to ask scoring for the round season-1/day-4")

	_, err = rounds(t, scoring).RankingAfter(t.Context(), "season-1/day-3")
	assert.ErrorIs(t, err, nations.ErrUnknownFlag)
}
