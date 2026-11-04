package rpc_planet_fronts

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Planet interface {
	GetTakesByCountry(
		ctx context.Context,
		req *connect.Request[planetv1.GetTakesByCountryRequest],
	) (*connect.Response[planetv1.GetTakesByCountryResponse], error)
}

const askTimeout = 5 * time.Second

type Reader struct {
	planet Planet
}

func New(planet Planet) *Reader {
	return &Reader{planet: planet}
}

func (r *Reader) Fronts(ctx context.Context, account cpsession.AccountID) (*progressionv1.GetFrontsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := r.planet.GetTakesByCountry(ctx, connect.NewRequest(&planetv1.GetTakesByCountryRequest{AccountId: account.String()}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask planet which countries the account took tiles for and from: %w", err)
	}
	return &progressionv1.GetFrontsResponse{
		PlaysFor:     countriesOf(res.Msg.GetTakenFor()),
		PlaysAgainst: countriesOf(res.Msg.GetTakenFrom()),
	}, nil
}

func countriesOf(takes []*planetv1.CountryTakes) []*progressionv1.CountryTiles {
	countries := make([]*progressionv1.CountryTiles, 0, len(takes))
	for _, take := range takes {
		countries = append(countries, &progressionv1.CountryTiles{CountryId: take.GetCountryId(), Tiles: take.GetTiles()})
	}
	return countries
}
