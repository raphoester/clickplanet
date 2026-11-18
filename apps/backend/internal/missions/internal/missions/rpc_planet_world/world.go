package rpc_planet_world

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
)

type Planet interface {
	GetTerritories(
		ctx context.Context,
		req *connect.Request[planetv1.GetTerritoriesRequest],
	) (*connect.Response[planetv1.GetTerritoriesResponse], error)
	GetCountrySizes(
		ctx context.Context,
		req *connect.Request[planetv1.GetCountrySizesRequest],
	) (*connect.Response[planetv1.GetCountrySizesResponse], error)
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
	sizes, err := r.planet.GetCountrySizes(ctx, connect.NewRequest(&planetv1.GetCountrySizesRequest{}))
	if err != nil {
		return missions.World{}, fmt.Errorf("failed to ask planet how big each country is: %w", err)
	}

	// The territories are counted on the board the map in play plays, which the sizes name too.
	scale, err := cpboards.ScaleOf(territories.Msg.GetBoard())
	if err != nil {
		return missions.World{}, fmt.Errorf("failed to scale the floors to planet's board: %w", err)
	}
	world := missions.World{Scale: scale}
	for _, held := range territories.Msg.GetTerritories() {
		territory := missions.Territory{Flag: held.GetCountryId(), Tiles: held.GetTiles(), Shields: held.GetShields()}
		for _, ground := range held.GetGrounds() {
			territory.Grounds = append(territory.Grounds, missions.Holding{Country: ground.GetCountryId(), Tiles: ground.GetTiles()})
		}
		world.Territories = append(world.Territories, territory)
	}
	for _, size := range sizes.Msg.GetCountrySizes() {
		world.CountrySizes = append(world.CountrySizes, missions.Holding{Country: size.GetCountryId(), Tiles: size.GetTiles()})
	}
	return world, nil
}
