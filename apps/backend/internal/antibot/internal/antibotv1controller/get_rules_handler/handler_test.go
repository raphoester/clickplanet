package get_rules_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

type stubSets struct {
	sets []*antibotv1.RuleSet
	err  error
}

func (s stubSets) Sets(context.Context) ([]*antibotv1.RuleSet, error) { return s.sets, s.err }

type playing cprules.SetName

func (p playing) Name() cprules.SetName { return cprules.SetName(p) }

func TestTheRulesAreEverySetKeptAndTheOnePlaying(t *testing.T) {
	sets := []*antibotv1.RuleSet{{Name: "default"}, {Name: "rush"}}

	res, err := get_rules_handler.New(stubSets{sets: sets}, playing("rush")).
		GetRules(t.Context(), connect.NewRequest(&antibotv1.GetRulesRequest{}))
	require.NoError(t, err)

	assert.True(t, proto.Equal(&antibotv1.GetRulesResponse{Sets: sets, Playing: "rush"}, res.Msg), "%v", res.Msg)
}
