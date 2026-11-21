package rpc_planet_territories_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/rpc_planet_territories"
)

type stubPlanet struct {
	planetv1connect.UnimplementedInternalServiceHandler

	answer *planetv1.GetTerritoriesResponse
}

func (s stubPlanet) GetTerritories(
	context.Context,
	*connect.Request[planetv1.GetTerritoriesRequest],
) (*connect.Response[planetv1.GetTerritoriesResponse], error) {
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
	assert.Equal(t, books.Snapshot{Tiles: 100, Held: map[books.Country]uint32{"bg": 7, "fr": 3}, Out: map[books.Country]time.Time{}}, snapshot)
}

func TestTheSnapshotSaysWhenEachFlagWasKnockedOut(t *testing.T) {
	at := time.Date(2026, 11, 21, 21, 40, 0, 0, time.UTC)
	planet := stubPlanet{answer: &planetv1.GetTerritoriesResponse{
		Tiles:       100,
		Territories: []*planetv1.Territory{{CountryId: "kp", Tiles: 14}},
		KnockedOut: []*planetv1.KnockedOutFlag{
			{CountryId: "ie", KnockedOutAt: timestamppb.New(at)},
			{CountryId: "pl", KnockedOutAt: timestamppb.New(at.Add(-time.Minute))},
		},
	}}

	snapshot, err := territories(t, planet).Snapshot(t.Context())

	require.NoError(t, err)
	assert.Equal(t, map[books.Country]time.Time{"ie": at, "pl": at.Add(-time.Minute)}, snapshot.Out)
}

func TestAKnockOutOfNoCountryIsAnError(t *testing.T) {
	planet := stubPlanet{answer: &planetv1.GetTerritoriesResponse{
		Tiles: 100, KnockedOut: []*planetv1.KnockedOutFlag{{KnockedOutAt: timestamppb.Now()}},
	}}

	_, err := territories(t, planet).Snapshot(t.Context())

	assert.ErrorIs(t, err, rpc_planet_territories.ErrNoCountry)
}

func TestTilesHeldByNoCountryAreAnError(t *testing.T) {
	planet := stubPlanet{answer: &planetv1.GetTerritoriesResponse{
		Tiles: 100, Territories: []*planetv1.Territory{{Tiles: 7}},
	}}

	_, err := territories(t, planet).Snapshot(t.Context())

	assert.ErrorIs(t, err, rpc_planet_territories.ErrNoCountry)
}
