package rpc_planet_fills_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar/rpc_planet_fills"
)

type planet struct {
	asked []*planetv1.FillChargesRequest
	err   error
}

func (p *planet) FillCharges(
	_ context.Context,
	req *connect.Request[planetv1.FillChargesRequest],
) (*connect.Response[planetv1.FillChargesResponse], error) {
	p.asked = append(p.asked, req.Msg)
	if p.err != nil {
		return nil, p.err
	}
	return connect.NewResponse(&planetv1.FillChargesResponse{}), nil
}

func TestOnlyTheStagesThatFillAskUnderARequestNamedForThePhase(t *testing.T) {
	p := &planet{}
	fills := rpc_planet_fills.New(p, rpc_planet_fills.Config{Finale: true})

	for _, stage := range []calendar.Stage{calendar.StageDays, calendar.StageFinale, calendar.StageOver} {
		require.NoError(t, fills.Play(t.Context(), calendar.Phase{Season: 0, Stage: stage}))
	}

	require.Len(t, p.asked, 1)
	assert.Equal(t, "season-0:finale:fill", p.asked[0].GetRequestId())
}

func TestEachStageFillsWhenTheConfigSaysSo(t *testing.T) {
	p := &planet{}
	fills := rpc_planet_fills.New(p, rpc_planet_fills.Config{Days: true, Finale: true, Over: true})

	for _, stage := range []calendar.Stage{calendar.StageDays, calendar.StageFinale, calendar.StageOver} {
		require.NoError(t, fills.Play(t.Context(), calendar.Phase{Season: 1, Stage: stage}))
	}

	requests := make([]string, 0, len(p.asked))
	for _, asked := range p.asked {
		requests = append(requests, asked.GetRequestId())
	}
	assert.Equal(t, []string{"season-1:days:fill", "season-1:finale:fill", "season-1:over:fill"}, requests)
}

func TestAnUnknownStageAsksNothing(t *testing.T) {
	p := &planet{}

	err := rpc_planet_fills.New(p, rpc_planet_fills.Config{Finale: true}).
		Play(t.Context(), calendar.Phase{Stage: "intermission"})

	require.ErrorIs(t, err, rpc_planet_fills.ErrUnknownStage)
	assert.Empty(t, p.asked)
}

func TestARefusalIsPassedOn(t *testing.T) {
	refused := connect.NewError(connect.CodeUnavailable, errors.New("planet is down"))

	err := rpc_planet_fills.New(&planet{err: refused}, rpc_planet_fills.Config{Finale: true}).
		Play(t.Context(), calendar.Phase{Stage: calendar.StageFinale})

	assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}
