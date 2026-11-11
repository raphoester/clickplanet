package rpc_rule_set_plays_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/rpc_rule_set_plays"
)

type planet struct {
	asked []*planetv1.PlayRuleSetRequest
	err   error
}

func (p *planet) PlayRuleSet(_ context.Context, req *connect.Request[planetv1.PlayRuleSetRequest],
) (*connect.Response[planetv1.PlayRuleSetResponse], error) {
	p.asked = append(p.asked, req.Msg)
	return connect.NewResponse(&planetv1.PlayRuleSetResponse{}), p.err
}

type antibot struct {
	asked []*antibotv1.PlayRuleSetRequest
	err   error
}

func (a *antibot) PlayRuleSet(_ context.Context, req *connect.Request[antibotv1.PlayRuleSetRequest],
) (*connect.Response[antibotv1.PlayRuleSetResponse], error) {
	a.asked = append(a.asked, req.Msg)
	return connect.NewResponse(&antibotv1.PlayRuleSetResponse{}), a.err
}

var decided = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func turnOf(set schedule.SetName) schedule.Turn {
	return schedule.New().With(schedule.Finale{Set: set, StartsAt: decided, EndsAt: decided.Add(time.Hour)}).TurnAt(decided)
}

func TestBothModulesPlayTheSetWithItsDecisionTime(t *testing.T) {
	toPlanet, toAntibot := &planet{}, &antibot{}

	require.NoError(t, rpc_rule_set_plays.New(toPlanet, toAntibot).Play(t.Context(), turnOf("storm"), decided))

	require.Len(t, toPlanet.asked, 1)
	require.Len(t, toAntibot.asked, 1)
	assert.Equal(t, "storm", toPlanet.asked[0].GetRuleSet())
	assert.Equal(t, decided, toPlanet.asked[0].GetDecidedAt().AsTime())
	assert.Equal(t, "storm", toAntibot.asked[0].GetRuleSet())
	assert.Equal(t, decided, toAntibot.asked[0].GetDecidedAt().AsTime())
}

func TestTheAntibotAloneLeavesPlanetOnItsSet(t *testing.T) {
	toPlanet, toAntibot := &planet{}, &antibot{}

	require.NoError(t, rpc_rule_set_plays.New(toPlanet, toAntibot).PlayAntibot(t.Context(), turnOf("storm"), decided))

	assert.Empty(t, toPlanet.asked)
	assert.Len(t, toAntibot.asked, 1)
}

func TestAModuleThatRefusesIsAnError(t *testing.T) {
	refused := connect.NewError(connect.CodeFailedPrecondition, errors.New("no such set"))

	err := rpc_rule_set_plays.New(&planet{err: refused}, &antibot{}).Play(t.Context(), turnOf("storm"), decided)

	require.ErrorIs(t, err, refused)
}
