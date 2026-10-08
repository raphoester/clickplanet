package rpc_planet_territory_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/rpc_planet_territory"
)

type stubPlanet struct {
	planetv1connect.UnimplementedInternalServiceHandler

	answer *planetv1.GetTerritoryResponse
	err    error
}

func (s stubPlanet) GetTerritory(
	context.Context,
	*connect.Request[planetv1.GetTerritoryRequest],
) (*connect.Response[planetv1.GetTerritoryResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.answer), nil
}

func territory(t *testing.T, planet stubPlanet) *rpc_planet_territory.Territory {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(planetv1connect.NewInternalServiceHandler(planet))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_planet_territory.New(planetv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestTheCensusIsWhatThePlanetModuleSaysEachCountryHolds(t *testing.T) {
	planet := stubPlanet{answer: &planetv1.GetTerritoryResponse{
		Tiles:       100,
		Territories: []*planetv1.Territory{{CountryId: "bg", Tiles: 7}, {CountryId: "fr", Tiles: 3}},
	}}

	census, err := territory(t, planet).Census(t.Context())

	require.NoError(t, err)
	assert.Equal(t, rounds.Census{Tiles: 100, Held: map[rounds.Country]uint32{"bg": 7, "fr": 3}}, census)
}

func TestTilesHeldByNoCountryAreAnError(t *testing.T) {
	planet := stubPlanet{answer: &planetv1.GetTerritoryResponse{
		Tiles: 100, Territories: []*planetv1.Territory{{Tiles: 7}},
	}}

	_, err := territory(t, planet).Census(t.Context())

	assert.ErrorIs(t, err, rpc_planet_territory.ErrNoCountry)
}

func TestAPlanetModuleThatFailsIsAnError(t *testing.T) {
	planet := stubPlanet{err: connect.NewError(connect.CodeInternal, errors.New("boom"))}

	_, err := territory(t, planet).Census(t.Context())

	assert.ErrorContains(t, err, "failed to ask the planet module")
}
