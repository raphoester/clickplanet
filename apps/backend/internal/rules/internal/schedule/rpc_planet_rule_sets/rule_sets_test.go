package rpc_planet_rule_sets_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/schedule/rpc_planet_rule_sets"
)

type stubPlanet struct {
	planetv1connect.UnimplementedInternalServiceHandler

	names []string
	err   error
}

func (s stubPlanet) GetRuleSetNames(
	context.Context,
	*connect.Request[planetv1.GetRuleSetNamesRequest],
) (*connect.Response[planetv1.GetRuleSetNamesResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&planetv1.GetRuleSetNamesResponse{Names: s.names}), nil
}

func ruleSets(t *testing.T, planet stubPlanet) *rpc_planet_rule_sets.RuleSets {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(planetv1connect.NewInternalServiceHandler(planet))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_planet_rule_sets.New(planetv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestASetPlanetKeepsIsKept(t *testing.T) {
	require.NoError(t, ruleSets(t, stubPlanet{names: []string{"default", "rush"}}).Keeps(t.Context(), "rush"))
}

func TestASetPlanetDoesNotKeepIsUnknownToPlanet(t *testing.T) {
	err := ruleSets(t, stubPlanet{names: []string{"default"}}).Keeps(t.Context(), "rush")

	require.ErrorIs(t, err, schedule.ErrUnknownSet)
	assert.Equal(t, `planet keeps no rule set of this name: "rush"`, err.Error())
}

func TestAPlanetModuleThatFailsIsAnError(t *testing.T) {
	err := ruleSets(t, stubPlanet{err: connect.NewError(connect.CodeInternal, errors.New("boom"))}).Keeps(t.Context(), "rush")

	require.ErrorContains(t, err, "failed to ask the planet module")
	assert.NotErrorIs(t, err, schedule.ErrUnknownSet)
}
