package rpc_planet_territories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/usecases/take_snapshot_usecase"
)

type Planet interface {
	GetTerritories(
		ctx context.Context,
		req *connect.Request[planetv1.GetTerritoriesRequest],
	) (*connect.Response[planetv1.GetTerritoriesResponse], error)
}

var ErrNoCountry = errors.New("the planet module answered tiles held by no country")

const askTimeout = 2 * time.Second

func New(planet Planet) *Territories {
	return &Territories{planet: planet}
}

type Territories struct {
	planet Planet
}

var _ take_snapshot_usecase.Territory = (*Territories)(nil)

func (t *Territories) Snapshot(ctx context.Context) (rounds.Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := t.planet.GetTerritories(ctx, connect.NewRequest(&planetv1.GetTerritoriesRequest{}))
	if err != nil {
		return rounds.Snapshot{}, fmt.Errorf("failed to ask the planet module what each country holds: %w", err)
	}

	held := make(map[rounds.Country]uint32, len(res.Msg.GetTerritories()))
	for _, territory := range res.Msg.GetTerritories() {
		if territory.GetCountryId() == "" {
			return rounds.Snapshot{}, ErrNoCountry
		}
		held[rounds.Country(territory.GetCountryId())] = territory.GetTiles()
	}
	return rounds.Snapshot{Tiles: res.Msg.GetTiles(), Held: held}, nil
}
