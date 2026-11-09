package rpc_planet_sieges_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/sieges/rpc_planet_sieges"
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

func TestASetPlaysASiegeWhenPlanetSaysSo(t *testing.T) {
	sieges := rpc_planet_sieges.New(planet{played: map[string]*planetv1.Rules{
		"capital-siege": {Siege: &planetv1.SiegeRules{On: new(true)}},
		"urf":           {Siege: &planetv1.SiegeRules{On: new(false)}},
	}})

	for set, want := range map[string]bool{"capital-siege": true, "urf": false, "gone": false} {
		siege, err := sieges.Siege(t.Context(), schedule.SetName(set))
		require.NoError(t, err, set)
		assert.Equal(t, want, siege, set)
	}
}

func TestPlanetDownIsAnError(t *testing.T) {
	_, err := rpc_planet_sieges.New(planet{err: connect.NewError(connect.CodeUnavailable, errors.New("down"))}).
		Siege(t.Context(), "capital-siege")

	require.Error(t, err)
}
