package rpc_bonuses_rule_sets_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1/bonusesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/rpc_bonuses_rule_sets"
)

type stubBonuses struct {
	bonusesv1connect.UnimplementedInternalServiceHandler

	names []string
	err   error
}

func (s stubBonuses) GetRuleSetNames(
	context.Context, *connect.Request[bonusesv1.GetRuleSetNamesRequest],
) (*connect.Response[bonusesv1.GetRuleSetNamesResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&bonusesv1.GetRuleSetNamesResponse{Names: s.names}), nil
}

func ruleSets(t *testing.T, bonuses stubBonuses) *rpc_bonuses_rule_sets.RuleSets {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(bonusesv1connect.NewInternalServiceHandler(bonuses))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_bonuses_rule_sets.New(bonusesv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestASetTheOffersKeepIsKept(t *testing.T) {
	require.NoError(t, ruleSets(t, stubBonuses{names: []string{"default", "rush"}}).Keeps(t.Context(), "rush"))
}

func TestASetTheOffersDoNotKeepIsUnknownToThem(t *testing.T) {
	err := ruleSets(t, stubBonuses{names: []string{"default"}}).Keeps(t.Context(), "rush")

	require.ErrorIs(t, err, schedule.ErrUnknownSet)
	assert.Equal(t, `the bonuses module keeps no rule set of this name: "rush"`, err.Error())
}

func TestOffersThatFailToAnswerAreAnError(t *testing.T) {
	err := ruleSets(t, stubBonuses{err: connect.NewError(connect.CodeInternal, errors.New("boom"))}).
		Keeps(t.Context(), "rush")

	require.ErrorContains(t, err, "failed to ask the bonuses module")
}
