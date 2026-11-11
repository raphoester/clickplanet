package get_rules_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules"
)

type stubSets struct {
	sets []*bonusesv1.RuleSet
	err  error
}

func (s stubSets) Sets(context.Context) ([]*bonusesv1.RuleSet, error) { return s.sets, s.err }

type playing rules.SetName

func (p playing) Name() rules.SetName { return rules.SetName(p) }

func TestTheRulesAreEverySetKeptAndTheOnePlaying(t *testing.T) {
	sets := []*bonusesv1.RuleSet{{Name: "default", Rules: rules.Seeded().Rules}, {Name: "rush"}}

	res, err := get_rules_handler.New(stubSets{sets: sets}, playing("default")).
		GetRules(t.Context(), connect.NewRequest(&bonusesv1.GetRulesRequest{}))
	require.NoError(t, err)

	assert.True(t, proto.Equal(&bonusesv1.GetRulesResponse{Sets: sets, Playing: "default"}, res.Msg), "%v", res.Msg)
}

func TestASetsReadThatFailsFailsTheCall(t *testing.T) {
	down := errors.New("postgres is down")

	_, err := get_rules_handler.New(stubSets{err: down}, playing("default")).
		GetRules(t.Context(), connect.NewRequest(&bonusesv1.GetRulesRequest{}))

	require.ErrorIs(t, err, down)
}
