package rpc_planet_arenas_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/rpc_planet_arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

type planet struct {
	played map[string]*planetv1.Rules
	err    error
	asked  []proto.Message
}

func (p *planet) OpenArena(_ context.Context, req *connect.Request[planetv1.OpenArenaRequest],
) (*connect.Response[planetv1.OpenArenaResponse], error) {
	p.asked = append(p.asked, req.Msg)
	return connect.NewResponse(&planetv1.OpenArenaResponse{}), p.err
}

func (p *planet) CloseArena(_ context.Context, req *connect.Request[planetv1.CloseArenaRequest],
) (*connect.Response[planetv1.CloseArenaResponse], error) {
	p.asked = append(p.asked, req.Msg)
	return connect.NewResponse(&planetv1.CloseArenaResponse{}), p.err
}

func (p *planet) GetRuleSet(_ context.Context, req *connect.Request[planetv1.GetRuleSetRequest],
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
	arenas := rpc_planet_arenas.New(&planet{played: map[string]*planetv1.Rules{
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
	_, err := rpc_planet_arenas.New(&planet{err: connect.NewError(connect.CodeUnavailable, errors.New("down"))}).
		Arena(t.Context(), "arena-battle")

	require.Error(t, err)
}

func TestAnOpenAndACloseAreEachOneCallInPlanetsWords(t *testing.T) {
	ends := time.Date(2026, 10, 30, 23, 0, 0, 0, time.UTC)
	asked := &planet{}
	arenas := rpc_planet_arenas.New(asked)

	require.NoError(t, arenas.Ask(t.Context(), moments.ArenaCommand{
		Action: moments.OpenArena, Arena: "battle-1", At: ends.Add(-2 * time.Hour), EndsAt: ends, Set: "arena-battle",
	}))
	require.NoError(t, arenas.Ask(t.Context(), moments.ArenaCommand{Action: moments.CloseArena, Arena: "battle-1", At: ends}))

	require.Len(t, asked.asked, 2)
	assert.True(t, proto.Equal(&planetv1.OpenArenaRequest{
		ArenaId: "battle-1", EndsAt: timestamppb.New(ends), RuleSet: "arena-battle",
	}, asked.asked[0]))
	assert.True(t, proto.Equal(&planetv1.CloseArenaRequest{ArenaId: "battle-1"}, asked.asked[1]))
}

func TestAnArenaPlanetRefusesIsRefusedForGood(t *testing.T) {
	for code, refused := range map[connect.Code]bool{
		connect.CodeFailedPrecondition: true, connect.CodeInvalidArgument: true, connect.CodeDeadlineExceeded: false,
	} {
		err := rpc_planet_arenas.New(&planet{err: connect.NewError(code, errors.New("no"))}).
			Ask(t.Context(), moments.ArenaCommand{Action: moments.CloseArena, Arena: "battle-1"})

		require.Error(t, err)
		assert.Equal(t, refused, errors.Is(err, moments.ErrRefused), "%s", code)
	}
}
