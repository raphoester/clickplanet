package rpc_planet_arenas_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/rpc_planet_arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

type planet struct {
	played map[string]*planetv1.Rules
	err    error
}

func (p planet) GetRuleSet(_ context.Context, req *connect.Request[planetv1.GetRuleSetRequest],
) (*connect.Response[planetv1.GetRuleSetResponse], error) {
	if p.err != nil {
		return nil, p.err
	}
	played, ok := p.played[req.Msg.GetName()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no such set"))
	}
	return connect.NewResponse(&planetv1.GetRuleSetResponse{Played: played}), nil
}

func TestASetPlaysAnArenaBattleWhenPlanetSaysSo(t *testing.T) {
	arenas := rpc_planet_arenas.New(planet{played: map[string]*planetv1.Rules{
		"arena-battle": {Arena: &planetv1.ArenaRules{On: new(true)}},
		"urf":          {Arena: &planetv1.ArenaRules{On: new(false)}},
	}})

	for set, want := range map[string]bool{"arena-battle": true, "urf": false, "gone": false} {
		inArena, err := arenas.Arena(t.Context(), schedule.SetName(set))
		require.NoError(t, err, set)
		assert.Equal(t, want, inArena, set)
	}
}

func TestPlanetDownIsAnError(t *testing.T) {
	_, err := rpc_planet_arenas.New(planet{err: connect.NewError(connect.CodeUnavailable, errors.New("down"))}).
		Arena(t.Context(), "arena-battle")

	require.Error(t, err)
}
