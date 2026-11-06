package rules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/rules"
)

func TestRulesAreTheirMessage(t *testing.T) {
	message := &missionsv1.Rules{FlagMissions: &missionsv1.FlagMissionRules{
		GoalPercent: 30, LeastGoal: 250, FewestPoints: 3, MostPoints: 10,
	}}

	kept, err := rules.RulesOf(message)

	require.NoError(t, err)
	assert.True(t, proto.Equal(message, kept.Snapshot()))
	assert.Equal(t, flagmissions.TestingSizing(), kept.FlagMissions())
}

func TestRulesThatCannotPlaySayWhichNumbers(t *testing.T) {
	_, err := rules.RulesOf(&missionsv1.Rules{FlagMissions: &missionsv1.FlagMissionRules{
		GoalPercent: 0, LeastGoal: 250, FewestPoints: 3, MostPoints: 10,
	}})

	require.ErrorIs(t, err, rules.ErrInvalidRules)
	assert.ErrorContains(t, err, "flagMissions: these numbers cannot size a flag mission: goalPercent is 0")
}

func TestNoFlagMissionNumbersCannotPlay(t *testing.T) {
	_, err := rules.RulesOf(&missionsv1.Rules{})

	assert.ErrorIs(t, err, rules.ErrInvalidRules)
}
