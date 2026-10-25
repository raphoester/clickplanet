package rpc_planet_world

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

type Planet interface {
	GetTerritories(
		ctx context.Context,
		req *connect.Request[planetv1.GetTerritoriesRequest],
	) (*connect.Response[planetv1.GetTerritoriesResponse], error)
	GetGrounds(
		ctx context.Context,
		req *connect.Request[planetv1.GetGroundsRequest],
	) (*connect.Response[planetv1.GetGroundsResponse], error)
}

const askTimeout = 5 * time.Second

func New(planet Planet) Reader {
	return Reader{planet: planet}
}

type Reader struct {
	planet Planet
}

var _ missions.Planet = Reader{}

func (r Reader) World(ctx context.Context) (missions.World, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	territories, err := r.planet.GetTerritories(ctx, connect.NewRequest(&planetv1.GetTerritoriesRequest{}))
	if err != nil {
		return missions.World{}, fmt.Errorf("failed to ask planet what each flag holds: %w", err)
	}
	grounds, err := r.planet.GetGrounds(ctx, connect.NewRequest(&planetv1.GetGroundsRequest{}))
	if err != nil {
		return missions.World{}, fmt.Errorf("failed to ask planet how big each country is: %w", err)
	}

	world := missions.World{}
	for _, held := range territories.Msg.GetTerritories() {
		world.Territories = append(world.Territories, missions.Holding{Country: held.GetCountryId(), Tiles: held.GetTiles()})
	}
	for _, ground := range grounds.Msg.GetGrounds() {
		world.Grounds = append(world.Grounds, missions.Holding{Country: ground.GetCountryId(), Tiles: ground.GetTiles()})
	}
	return world, nil
}
