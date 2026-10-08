package rpc_planet_territory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/usecases/take_census_usecase"
)

type Planet interface {
	GetTerritory(
		ctx context.Context,
		req *connect.Request[planetv1.GetTerritoryRequest],
	) (*connect.Response[planetv1.GetTerritoryResponse], error)
}

var ErrNoCountry = errors.New("the planet module answered tiles held by no country")

const askTimeout = 2 * time.Second

func New(planet Planet) *Territory {
	return &Territory{planet: planet}
}

type Territory struct {
	planet Planet
}

var _ take_census_usecase.Territory = (*Territory)(nil)

func (t *Territory) Census(ctx context.Context) (rounds.Census, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := t.planet.GetTerritory(ctx, connect.NewRequest(&planetv1.GetTerritoryRequest{}))
	if err != nil {
		return rounds.Census{}, fmt.Errorf("failed to ask the planet module what each country holds: %w", err)
	}

	held := make(map[rounds.Country]uint32, len(res.Msg.GetTerritories()))
	for _, territory := range res.Msg.GetTerritories() {
		if territory.GetCountryId() == "" {
			return rounds.Census{}, ErrNoCountry
		}
		held[rounds.Country(territory.GetCountryId())] = territory.GetTiles()
	}
	return rounds.Census{Tiles: res.Msg.GetTiles(), Held: held}, nil
}
