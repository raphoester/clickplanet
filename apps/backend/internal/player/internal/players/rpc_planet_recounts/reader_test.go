package rpc_planet_recounts_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/rpc_planet_recounts"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubPlanet struct {
	answer *planetv1.GetBombsAndFortifiesResponse
	asked  []string
}

func (s *stubPlanet) GetBombsAndFortifies(
	_ context.Context,
	req *connect.Request[planetv1.GetBombsAndFortifiesRequest],
) (*connect.Response[planetv1.GetBombsAndFortifiesResponse], error) {
	s.asked = req.Msg.GetAccountIds()
	return connect.NewResponse(s.answer), nil
}

func TestEachAccountPlanetAnsweredForIsARecount(t *testing.T) {
	planet := &stubPlanet{answer: &planetv1.GetBombsAndFortifiesResponse{Accounts: []*planetv1.BombsAndFortifies{
		{AccountId: ada, BombsDropped: 3, TilesFortified: 52},
	}}}
	id, err := players.AccountIDOf(ada)
	require.NoError(t, err)

	recounts, err := rpc_planet_recounts.New(planet).Recounts(t.Context(), []players.AccountID{id})

	require.NoError(t, err)
	assert.Equal(t, []string{ada}, planet.asked)
	assert.Equal(t, map[players.AccountID]players.Recount{id: {BombsDropped: 3, TilesFortified: 52}}, recounts)
}

func TestAnAnswerForAnIDThatIsNoAccountIsAnError(t *testing.T) {
	planet := &stubPlanet{answer: &planetv1.GetBombsAndFortifiesResponse{Accounts: []*planetv1.BombsAndFortifies{{AccountId: "nope"}}}}

	_, err := rpc_planet_recounts.New(planet).Recounts(t.Context(), nil)

	assert.ErrorIs(t, err, players.ErrInvalidAccount)
}
