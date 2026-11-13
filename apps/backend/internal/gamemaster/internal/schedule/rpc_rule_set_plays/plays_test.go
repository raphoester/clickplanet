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
	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
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

type bonuses struct {
	asked []*bonusesv1.PlayRuleSetRequest
	err   error
}

func (b *bonuses) PlayRuleSet(_ context.Context, req *connect.Request[bonusesv1.PlayRuleSetRequest],
) (*connect.Response[bonusesv1.PlayRuleSetResponse], error) {
	b.asked = append(b.asked, req.Msg)
	return connect.NewResponse(&bonusesv1.PlayRuleSetResponse{}), b.err
}

type quizzes struct {
	asked []*quizzesv1.PlayRuleSetRequest
	err   error
}

func (q *quizzes) PlayRuleSet(_ context.Context, req *connect.Request[quizzesv1.PlayRuleSetRequest],
) (*connect.Response[quizzesv1.PlayRuleSetResponse], error) {
	q.asked = append(q.asked, req.Msg)
	return connect.NewResponse(&quizzesv1.PlayRuleSetResponse{}), q.err
}

var decided = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func turnOf(set schedule.SetName) schedule.Turn {
	return schedule.New().With(schedule.Finale{Set: set, StartsAt: decided, EndsAt: decided.Add(time.Hour)}).TurnAt(decided)
}

func TestEveryModulePlaysTheSetWithItsDecisionTime(t *testing.T) {
	toPlanet, toAntibot, toBonuses, toQuizzes := &planet{}, &antibot{}, &bonuses{}, &quizzes{}

	require.NoError(t, rpc_rule_set_plays.New(toPlanet, toAntibot, toBonuses, toQuizzes).
		Play(t.Context(), turnOf("storm"), decided))

	require.Len(t, toPlanet.asked, 1)
	require.Len(t, toAntibot.asked, 1)
	require.Len(t, toBonuses.asked, 1)
	require.Len(t, toQuizzes.asked, 1)
	assert.Equal(t, "storm", toBonuses.asked[0].GetRuleSet())
	assert.Equal(t, decided, toBonuses.asked[0].GetDecidedAt().AsTime())
	assert.Equal(t, "storm", toQuizzes.asked[0].GetRuleSet())
	assert.Equal(t, decided, toQuizzes.asked[0].GetDecidedAt().AsTime())
	assert.Equal(t, "storm", toPlanet.asked[0].GetRuleSet())
	assert.Equal(t, decided, toPlanet.asked[0].GetDecidedAt().AsTime())
	assert.Equal(t, "storm", toAntibot.asked[0].GetRuleSet())
	assert.Equal(t, decided, toAntibot.asked[0].GetDecidedAt().AsTime())
}

func TestTheAntibotAloneLeavesPlanetTheOffersAndTheQuizzesOnTheirSet(t *testing.T) {
	toPlanet, toAntibot, toBonuses, toQuizzes := &planet{}, &antibot{}, &bonuses{}, &quizzes{}

	require.NoError(t, rpc_rule_set_plays.New(toPlanet, toAntibot, toBonuses, toQuizzes).
		PlayAntibot(t.Context(), turnOf("storm"), decided))

	assert.Empty(t, toPlanet.asked)
	assert.Empty(t, toBonuses.asked, "the offers wait for the battle's map with planet")
	assert.Empty(t, toQuizzes.asked, "and so do the quizzes")
	assert.Len(t, toAntibot.asked, 1)
}

func TestAModuleThatRefusesIsAnError(t *testing.T) {
	refused := connect.NewError(connect.CodeFailedPrecondition, errors.New("no such set"))

	err := rpc_rule_set_plays.New(&planet{err: refused}, &antibot{}, &bonuses{}, &quizzes{}).
		Play(t.Context(), turnOf("storm"), decided)

	require.ErrorIs(t, err, refused)

	err = rpc_rule_set_plays.New(&planet{}, &antibot{}, &bonuses{err: refused}, &quizzes{}).
		Play(t.Context(), turnOf("storm"), decided)

	require.ErrorIs(t, err, refused, "the offers' refusal is the schedule's too: it calls again")

	err = rpc_rule_set_plays.New(&planet{}, &antibot{}, &bonuses{}, &quizzes{err: refused}).
		Play(t.Context(), turnOf("storm"), decided)

	require.ErrorIs(t, err, refused, "and so is the quizzes'")
}
