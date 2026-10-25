package get_rules_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

type stubSets []rules.Set

func (s stubSets) Sets() []rules.Set { return s }

type stubChanges struct {
	changes []*planetv1.RuleChange
	err     error
}

func (s stubChanges) Changes(context.Context) ([]*planetv1.RuleChange, error) {
	return s.changes, s.err
}

func TestTheRulesAreEverySetWithItsValuesAndTheChanges(t *testing.T) {
	sets := stubSets{
		{
			Name: rules.Default, Tempo: 1, AntiBot: rules.AntiBot{Crowd: 6},
			Boxes: bonuses.Pace{MinInterval: 4 * time.Minute, MaxInterval: 8 * time.Minute, MaxChargesPerHour: 12},
		},
		{
			Name: "rush", Tempo: 3, AntiBot: rules.AntiBot{Crowd: 50},
			Boxes: bonuses.Pace{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30},
		},
		{
			Name: "frozen", Tempo: 1, AntiBot: rules.AntiBot{Crowd: 6}, Frozen: true,
			Boxes: bonuses.Pace{MinInterval: 4 * time.Minute, MaxInterval: 8 * time.Minute, MaxChargesPerHour: 12},
		},
	}
	changes := []*planetv1.RuleChange{{RequestId: "season-0:finale", RuleSet: "rush"}}

	res, err := get_rules_handler.New(sets, stubChanges{changes: changes}).
		GetRules(t.Context(), connect.NewRequest(&planetv1.GetRulesRequest{}))
	require.NoError(t, err)

	want := &planetv1.GetRulesResponse{
		Sets: []*planetv1.RuleSet{
			{Name: "default", Tempo: 1, Boxes: &planetv1.BoxPace{
				MinInterval: durationpb.New(4 * time.Minute), MaxInterval: durationpb.New(8 * time.Minute), MaxChargesPerHour: 12,
			}, AntiBot: &planetv1.AntiBotRules{Crowd: 6}},
			{Name: "rush", Tempo: 3, Boxes: &planetv1.BoxPace{
				MinInterval: durationpb.New(time.Minute), MaxInterval: durationpb.New(2 * time.Minute), MaxChargesPerHour: 30,
			}, AntiBot: &planetv1.AntiBotRules{Crowd: 50}},
			{Name: "frozen", Tempo: 1, Boxes: &planetv1.BoxPace{
				MinInterval: durationpb.New(4 * time.Minute), MaxInterval: durationpb.New(8 * time.Minute), MaxChargesPerHour: 12,
			}, AntiBot: &planetv1.AntiBotRules{Crowd: 6}, Frozen: true},
		},
		Changes: changes,
	}
	assert.True(t, proto.Equal(want, res.Msg), "%v", res.Msg)
}

func TestAChangesReadThatFailsFailsTheCall(t *testing.T) {
	down := errors.New("postgres is down")

	_, err := get_rules_handler.New(stubSets{}, stubChanges{err: down}).
		GetRules(t.Context(), connect.NewRequest(&planetv1.GetRulesRequest{}))

	require.ErrorIs(t, err, down)
}
