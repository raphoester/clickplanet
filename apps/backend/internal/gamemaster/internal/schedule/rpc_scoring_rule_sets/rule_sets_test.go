package rpc_scoring_rule_sets_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/rpc_scoring_rule_sets"
)

type stubScoring struct {
	scoringv1connect.UnimplementedInternalServiceHandler

	names []string
	err   error
}

func (s stubScoring) GetRuleSetNames(
	context.Context,
	*connect.Request[scoringv1.GetRuleSetNamesRequest],
) (*connect.Response[scoringv1.GetRuleSetNamesResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&scoringv1.GetRuleSetNamesResponse{Names: s.names}), nil
}

func ruleSets(t *testing.T, scoring stubScoring) *rpc_scoring_rule_sets.RuleSets {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(scoringv1connect.NewInternalServiceHandler(scoring))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_scoring_rule_sets.New(scoringv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestASetScoringKeepsIsKept(t *testing.T) {
	require.NoError(t, ruleSets(t, stubScoring{names: []string{"default", "rush"}}).Keeps(t.Context(), "rush"))
}

func TestASetScoringDoesNotKeepIsUnknown(t *testing.T) {
	err := ruleSets(t, stubScoring{names: []string{"default"}}).Keeps(t.Context(), "rush")

	require.ErrorIs(t, err, schedule.ErrUnknownSet)
	assert.Equal(t, `scoring keeps no rule set of this name: "rush"`, err.Error())
}

func TestAScoringModuleThatFailsIsAnError(t *testing.T) {
	err := ruleSets(t, stubScoring{err: connect.NewError(connect.CodeInternal, errors.New("boom"))}).Keeps(t.Context(), "rush")

	require.ErrorContains(t, err, "failed to ask the scoring module")
	assert.NotErrorIs(t, err, schedule.ErrUnknownSet)
}
