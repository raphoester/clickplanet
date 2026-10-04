package rpc_planet_shares_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/lead/rpc_planet_shares"
)

type stubPlanet struct {
	planetv1connect.UnimplementedInternalServiceHandler

	res *planetv1.GetSharesResponse
	err error
}

func (s stubPlanet) GetShares(
	context.Context,
	*connect.Request[planetv1.GetSharesRequest],
) (*connect.Response[planetv1.GetSharesResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.res), nil
}

func serve(t *testing.T, planet stubPlanet) planetv1connect.InternalServiceClient {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(planetv1connect.NewInternalServiceHandler(planet))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return planetv1connect.NewInternalServiceClient(server.Client(), server.URL)
}

func TestTheSharesAreWhatPlanetHolds(t *testing.T) {
	planet := stubPlanet{res: &planetv1.GetSharesResponse{MapTiles: 262119, Countries: []*planetv1.CountryTiles{
		{Country: "dz", Tiles: 40000}, {Country: "fr", Tiles: 39000},
	}}}

	shares, err := rpc_planet_shares.New(serve(t, planet)).Shares(t.Context())

	require.NoError(t, err)
	assert.Equal(t, uint32(40000), shares.Tiles("dz"))
	assert.Equal(t, uint32(39000), shares.Tiles("fr"))
	leader, ok := shares.Leader()
	require.True(t, ok)
	assert.Equal(t, "dz", leader)
}

func TestAFailureToAskIsAnError(t *testing.T) {
	_, err := rpc_planet_shares.New(serve(t, stubPlanet{err: errors.New("down")})).Shares(t.Context())
	require.Error(t, err)
}
