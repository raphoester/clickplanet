package get_rules_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rules_handler"
)

type stubSets struct {
	sets []*planetv1.RuleSet
	err  error
}

func (s stubSets) Sets(context.Context) ([]*planetv1.RuleSet, error) { return s.sets, s.err }

func TestTheRulesAreEverySetKept(t *testing.T) {
	sets := []*planetv1.RuleSet{{Name: "default", Tempo: 1}, {Name: "rush", Tempo: 3}}

	res, err := get_rules_handler.New(stubSets{sets: sets}).GetRules(t.Context(), connect.NewRequest(&planetv1.GetRulesRequest{}))
	require.NoError(t, err)

	assert.True(t, proto.Equal(&planetv1.GetRulesResponse{Sets: sets}, res.Msg), "%v", res.Msg)
}

func TestASetsReadThatFailsFailsTheCall(t *testing.T) {
	down := errors.New("postgres is down")

	_, err := get_rules_handler.New(stubSets{err: down}).GetRules(t.Context(), connect.NewRequest(&planetv1.GetRulesRequest{}))

	require.ErrorIs(t, err, down)
}
