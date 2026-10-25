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

type stubLayers struct {
	baselines []*planetv1.RuleBaseline
	override  *planetv1.RuleOverride
	err       error
}

func (s stubLayers) Baselines(context.Context) ([]*planetv1.RuleBaseline, error) {
	return s.baselines, s.err
}

func (s stubLayers) Override(context.Context) (*planetv1.RuleOverride, error) {
	return s.override, s.err
}

func TestTheRulesAreEverySetKeptAndTheLayers(t *testing.T) {
	sets := []*planetv1.RuleSet{{Name: "default", Tempo: 1}, {Name: "rush", Tempo: 3}}
	layers := stubLayers{
		baselines: []*planetv1.RuleBaseline{{RequestId: "season-0:finale", RuleSet: "rush"}},
		override:  &planetv1.RuleOverride{RuleSet: "default"},
	}

	res, err := get_rules_handler.New(stubSets{sets: sets}, layers).
		GetRules(t.Context(), connect.NewRequest(&planetv1.GetRulesRequest{}))
	require.NoError(t, err)

	assert.True(t, proto.Equal(&planetv1.GetRulesResponse{
		Sets: sets, Baselines: layers.baselines, Override: layers.override,
	}, res.Msg), "%v", res.Msg)
}

func TestAReadThatFailsFailsTheCall(t *testing.T) {
	down := errors.New("postgres is down")

	_, err := get_rules_handler.New(stubSets{err: down}, stubLayers{}).
		GetRules(t.Context(), connect.NewRequest(&planetv1.GetRulesRequest{}))
	require.ErrorIs(t, err, down)

	_, err = get_rules_handler.New(stubSets{}, stubLayers{err: down}).
		GetRules(t.Context(), connect.NewRequest(&planetv1.GetRulesRequest{}))
	require.ErrorIs(t, err, down)
}
