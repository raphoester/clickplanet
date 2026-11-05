package rpc_planet_allegiance_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/rpc_planet_allegiance"
)

type stubPlanet struct {
	asked  []string
	answer *planetv1.GetAllegianceOfResponse
	err    error
}

func (p *stubPlanet) GetAllegianceOf(
	_ context.Context, req *connect.Request[planetv1.GetAllegianceOfRequest],
) (*connect.Response[planetv1.GetAllegianceOfResponse], error) {
	p.asked = append(p.asked, req.Msg.GetAccountId())
	if p.err != nil {
		return nil, p.err
	}
	return connect.NewResponse(p.answer), nil
}

var account = missions.AccountID{15: 7}

func TestTheFlagIsThePledgedOne(t *testing.T) {
	planet := &stubPlanet{answer: &planetv1.GetAllegianceOfResponse{Allegiance: &planetv1.Allegiance{CountryId: "fr"}}}

	flag, err := rpc_planet_allegiance.New(planet).FlagOf(t.Context(), account)

	require.NoError(t, err)
	assert.Equal(t, "fr", flag)
	assert.Equal(t, []string{account.String()}, planet.asked)
}

func TestAnAccountThatNeverPledgedHasNoFlag(t *testing.T) {
	flag, err := rpc_planet_allegiance.New(&stubPlanet{answer: &planetv1.GetAllegianceOfResponse{}}).FlagOf(t.Context(), account)

	require.NoError(t, err)
	assert.Empty(t, flag)
}

func TestAPlanetThatFailsIsAnError(t *testing.T) {
	_, err := rpc_planet_allegiance.New(&stubPlanet{err: errors.New("down")}).FlagOf(t.Context(), account)

	assert.Error(t, err)
}
