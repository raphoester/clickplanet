package rpc_planet_rules_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar/rpc_planet_rules"
)

type planet struct {
	asked []*planetv1.ApplyRuleSetRequest
	err   error
}

func (p *planet) ApplyRuleSet(
	_ context.Context,
	req *connect.Request[planetv1.ApplyRuleSetRequest],
) (*connect.Response[planetv1.ApplyRuleSetResponse], error) {
	p.asked = append(p.asked, req.Msg)
	if p.err != nil {
		return nil, p.err
	}
	return connect.NewResponse(&planetv1.ApplyRuleSetResponse{}), nil
}

func TestEachStageAsksForItsSetUnderARequestNamedForThePhase(t *testing.T) {
	p := &planet{}
	rules := rpc_planet_rules.New(p, rpc_planet_rules.Config{Days: "calm", Finale: "rush", Over: "frozen"})

	for _, stage := range []calendar.Stage{calendar.StageDays, calendar.StageFinale, calendar.StageOver} {
		require.NoError(t, rules.Play(t.Context(), calendar.Phase{Season: 0, Stage: stage}))
	}

	require.Len(t, p.asked, 3)
	assert.Equal(t, "season-0:days", p.asked[0].GetRequestId())
	assert.Equal(t, "calm", p.asked[0].GetRuleSet())
	assert.Equal(t, "season-0:finale", p.asked[1].GetRequestId())
	assert.Equal(t, "rush", p.asked[1].GetRuleSet())
	assert.Equal(t, "season-0:over", p.asked[2].GetRequestId())
	assert.Equal(t, "frozen", p.asked[2].GetRuleSet())
}

func TestAStageWithNoSetPlaysPlanetsDefault(t *testing.T) {
	p := &planet{}

	require.NoError(t, rpc_planet_rules.New(p, rpc_planet_rules.Config{Finale: "rush"}).
		Play(t.Context(), calendar.Phase{Season: 1, Stage: calendar.StageDays}))

	require.Len(t, p.asked, 1)
	assert.Equal(t, "season-1:days", p.asked[0].GetRequestId())
	assert.Equal(t, "default", p.asked[0].GetRuleSet())
}

func TestAnUnknownStageAsksNothing(t *testing.T) {
	p := &planet{}

	err := rpc_planet_rules.New(p, rpc_planet_rules.Config{}).Play(t.Context(), calendar.Phase{Stage: "intermission"})

	require.ErrorIs(t, err, rpc_planet_rules.ErrUnknownStage)
	assert.Empty(t, p.asked)
}

func TestARefusalIsPassedOn(t *testing.T) {
	refused := connect.NewError(connect.CodeInvalidArgument, errors.New("no such set"))

	err := rpc_planet_rules.New(&planet{err: refused}, rpc_planet_rules.Config{Finale: "rush"}).
		Play(t.Context(), calendar.Phase{Stage: calendar.StageFinale})

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
