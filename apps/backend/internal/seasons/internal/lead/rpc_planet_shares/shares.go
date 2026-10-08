package rpc_planet_shares

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/lead"
)

type Planet interface {
	GetTerritories(
		ctx context.Context,
		req *connect.Request[planetv1.GetTerritoriesRequest],
	) (*connect.Response[planetv1.GetTerritoriesResponse], error)
}

const askTimeout = 2 * time.Second

func New(planet Planet) *Shares {
	return &Shares{planet: planet}
}

type Shares struct {
	planet Planet
}

func (s *Shares) Shares(ctx context.Context) (lead.Shares, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := s.planet.GetTerritories(ctx, connect.NewRequest(&planetv1.GetTerritoriesRequest{}))
	if err != nil {
		return lead.Shares{}, fmt.Errorf("failed to call planet.v1.InternalService/GetTerritories: %w", err)
	}

	tiles := make(map[string]uint32, len(res.Msg.GetTerritories()))
	for _, territory := range res.Msg.GetTerritories() {
		tiles[territory.GetCountryId()] = territory.GetTiles()
	}

	return lead.SharesOf(tiles), nil
}
