package rpc_planet_fronts_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/rpc_planet_fronts"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type fakePlanet struct {
	answer *planetv1.GetTakesByCountryResponse
	err    error
	asked  []string
}

func (f *fakePlanet) GetTakesByCountry(
	_ context.Context,
	req *connect.Request[planetv1.GetTakesByCountryRequest],
) (*connect.Response[planetv1.GetTakesByCountryResponse], error) {
	f.asked = append(f.asked, req.Msg.GetAccountId())
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.answer), nil
}

func account(t *testing.T) players.AccountID {
	t.Helper()

	id, err := players.AccountIDOf(ada)
	require.NoError(t, err)
	return id
}

func TestTheFlagsTilesWereTakenForAndFromAreTheCountriesPlayedForAndAgainstInTheirOrder(t *testing.T) {
	planet := &fakePlanet{answer: &planetv1.GetTakesByCountryResponse{
		TakenFor:  []*planetv1.CountryTakes{{CountryId: "fr", Tiles: 5}, {CountryId: "it", Tiles: 2}},
		TakenFrom: []*planetv1.CountryTakes{{CountryId: "de", Tiles: 3}},
	}}

	fronts, err := rpc_planet_fronts.New(planet).Fronts(t.Context(), account(t))

	require.NoError(t, err)
	assert.True(t, proto.Equal(&playerv1.GetFrontsResponse{
		PlaysFor:     []*playerv1.CountryTiles{{CountryId: "fr", Tiles: 5}, {CountryId: "it", Tiles: 2}},
		PlaysAgainst: []*playerv1.CountryTiles{{CountryId: "de", Tiles: 3}},
	}, fronts))
	assert.Equal(t, []string{ada}, planet.asked)
}

func TestAFailureToAskPlanetIsAnError(t *testing.T) {
	planet := &fakePlanet{err: errors.New("planet is down")}

	_, err := rpc_planet_fronts.New(planet).Fronts(t.Context(), account(t))

	assert.ErrorIs(t, err, planet.err)
}
