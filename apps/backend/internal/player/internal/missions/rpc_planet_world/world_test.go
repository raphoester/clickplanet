package rpc_planet_world_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions/rpc_planet_world"
)

type stubPlanet struct {
	territories []*planetv1.Territory
	grounds     []*planetv1.Ground
	err         error
}

func (p stubPlanet) GetTerritories(
	context.Context, *connect.Request[planetv1.GetTerritoriesRequest],
) (*connect.Response[planetv1.GetTerritoriesResponse], error) {
	if p.err != nil {
		return nil, p.err
	}
	return connect.NewResponse(&planetv1.GetTerritoriesResponse{Territories: p.territories}), nil
}

func (p stubPlanet) GetGrounds(
	context.Context, *connect.Request[planetv1.GetGroundsRequest],
) (*connect.Response[planetv1.GetGroundsResponse], error) {
	return connect.NewResponse(&planetv1.GetGroundsResponse{Grounds: p.grounds}), nil
}

func TestTheWorldIsWhatEachFlagHoldsAndHowBigEachCountryIs(t *testing.T) {
	world, err := rpc_planet_world.New(stubPlanet{
		territories: []*planetv1.Territory{{CountryId: "dz", Tiles: 9}},
		grounds:     []*planetv1.Ground{{CountryId: "br", Tiles: 900}},
	}).World(t.Context())

	require.NoError(t, err)
	assert.Equal(t, missions.World{
		Territories: []missions.Holding{{Country: "dz", Tiles: 9}},
		Grounds:     []missions.Holding{{Country: "br", Tiles: 900}},
	}, world)
}

func TestAPlanetThatFailsIsAnError(t *testing.T) {
	_, err := rpc_planet_world.New(stubPlanet{err: errors.New("down")}).World(t.Context())

	assert.Error(t, err)
}
