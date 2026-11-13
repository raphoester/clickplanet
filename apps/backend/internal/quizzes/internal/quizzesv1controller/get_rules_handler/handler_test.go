package get_rules_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

type stubSets struct {
	sets []*quizzesv1.RuleSet
	err  error
}

func (s stubSets) Sets(context.Context) ([]*quizzesv1.RuleSet, error) { return s.sets, s.err }

type playing cprules.SetName

func (p playing) Name() cprules.SetName { return cprules.SetName(p) }

func TestTheRulesAreEverySetKeptAndTheOnePlaying(t *testing.T) {
	sets := []*quizzesv1.RuleSet{{Name: "default", Rules: rules.Seeded().Rules}, {Name: "rush"}}

	res, err := get_rules_handler.New(stubSets{sets: sets}, playing("default")).
		GetRules(t.Context(), connect.NewRequest(&quizzesv1.GetRulesRequest{}))
	require.NoError(t, err)

	assert.True(t, proto.Equal(&quizzesv1.GetRulesResponse{Sets: sets, Playing: "default"}, res.Msg), "%v", res.Msg)
}
