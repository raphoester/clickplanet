package get_rules_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

type stubSets struct {
	sets []*boxesv1.RuleSet
	err  error
}

func (s stubSets) Sets(context.Context) ([]*boxesv1.RuleSet, error) { return s.sets, s.err }

type playing cprules.SetName

func (p playing) Name() cprules.SetName { return cprules.SetName(p) }

func TestTheRulesAreEverySetKeptAndTheOnePlaying(t *testing.T) {
	sets := []*boxesv1.RuleSet{{Name: "default", Rules: rules.Seeded().Rules}, {Name: "rush"}}

	res, err := get_rules_handler.New(stubSets{sets: sets}, playing("default")).
		GetRules(t.Context(), connect.NewRequest(&boxesv1.GetRulesRequest{}))
	require.NoError(t, err)

	assert.True(t, proto.Equal(&boxesv1.GetRulesResponse{Sets: sets, Playing: "default"}, res.Msg), "%v", res.Msg)
}
