package bonuses

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestALevelUpFillsEveryKindWeighedWhileTheGatesAreOff(t *testing.T) {
	stacks, ok := LevelUp{Account: "ada", Grant: GrantID{15: 1}}.Stacks(TestRules())

	require.True(t, ok)
	assert.Equal(t, Stacks{
		Grant: GrantID{15: 1}, Account: "ada",
		Kinds: []Kind{KindRefill, KindSpreadClicks, KindBomb, KindEncloseClicks, KindShields},
	}, stacks, "the virus, the mine and the rally weigh 0")
}

func TestALevelUpFillsTheOpenKindsAndTheOnesItsLevelUnlockedWhileTheGatesAreOn(t *testing.T) {
	stacks, ok := LevelUp{Unlocked: Unlocked{KindEncloseClicks}}.Stacks(TestGated())

	require.True(t, ok)
	assert.Equal(t, []Kind{KindRefill, KindSpreadClicks, KindEncloseClicks, KindShields}, stacks.Kinds,
		"level 5 unlocks no bomb")
}

func TestALevelUpWhoseKindsAllWeighNothingFillsNothing(t *testing.T) {
	rules := TestGated()
	rules.Kinds = map[Kind]float64{KindBomb: 1}

	_, ok := LevelUp{}.Stacks(rules)

	assert.False(t, ok, "the bomb is not unlocked, and nothing else weighs")
}

func TestAGrantIdThatIsNoIdIsRefused(t *testing.T) {
	for _, bad := range []string{"", "a-token", "00000000-0000-0000-0000-000000000000"} {
		_, err := GrantIDOf(bad)
		require.ErrorIs(t, err, ErrInvalidGrant, "%q", bad)
	}
}
