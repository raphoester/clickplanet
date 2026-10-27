package get_rules_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_rules_handler"
)

type stubSets struct {
	sets []*seasonsv1.RuleSet
	err  error
}

func (s stubSets) Sets(context.Context) ([]*seasonsv1.RuleSet, error) { return s.sets, s.err }

type playing rules.SetName

func (p playing) Name() rules.SetName { return rules.SetName(p) }

func TestTheRulesAreEverySetKeptAndTheOnePlaying(t *testing.T) {
	sets := []*seasonsv1.RuleSet{{Name: "default"}, {Name: "rush"}}

	res, err := get_rules_handler.New(stubSets{sets: sets}, playing("rush")).
		GetRules(t.Context(), connect.NewRequest(&seasonsv1.GetRulesRequest{}))
	require.NoError(t, err)

	assert.True(t, proto.Equal(&seasonsv1.GetRulesResponse{Sets: sets, Playing: "rush"}, res.Msg), "%v", res.Msg)
}

func TestASetsReadThatFailsFailsTheCall(t *testing.T) {
	down := errors.New("postgres is down")

	_, err := get_rules_handler.New(stubSets{err: down}, playing("default")).
		GetRules(t.Context(), connect.NewRequest(&seasonsv1.GetRulesRequest{}))

	require.ErrorIs(t, err, down)
}
