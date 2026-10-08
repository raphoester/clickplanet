package rpc_planet_territories_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/rpc_planet_territories"
)

type stubPlanet struct {
	planetv1connect.UnimplementedInternalServiceHandler

	answer *planetv1.GetTerritoriesResponse
	err    error
}

func (s stubPlanet) GetTerritories(
	context.Context,
	*connect.Request[planetv1.GetTerritoriesRequest],
) (*connect.Response[planetv1.GetTerritoriesResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.answer), nil
}

func territories(t *testing.T, planet stubPlanet) *rpc_planet_territories.Territories {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(planetv1connect.NewInternalServiceHandler(planet))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_planet_territories.New(planetv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestTheSnapshotIsWhatThePlanetModuleSaysEachCountryHolds(t *testing.T) {
	planet := stubPlanet{answer: &planetv1.GetTerritoriesResponse{
		Tiles:       100,
		Territories: []*planetv1.Territory{{CountryId: "bg", Tiles: 7}, {CountryId: "fr", Tiles: 3}},
	}}

	snapshot, err := territories(t, planet).Snapshot(t.Context())

	require.NoError(t, err)
	assert.Equal(t, rounds.Snapshot{Tiles: 100, Held: map[rounds.Country]uint32{"bg": 7, "fr": 3}}, snapshot)
}

func TestTilesHeldByNoCountryAreAnError(t *testing.T) {
	planet := stubPlanet{answer: &planetv1.GetTerritoriesResponse{
		Tiles: 100, Territories: []*planetv1.Territory{{Tiles: 7}},
	}}

	_, err := territories(t, planet).Snapshot(t.Context())

	assert.ErrorIs(t, err, rpc_planet_territories.ErrNoCountry)
}

func TestAPlanetModuleThatFailsIsAnError(t *testing.T) {
	planet := stubPlanet{err: connect.NewError(connect.CodeInternal, errors.New("boom"))}

	_, err := territories(t, planet).Snapshot(t.Context())

	assert.ErrorContains(t, err, "failed to ask the planet module")
}
