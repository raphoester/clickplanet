package bonuses

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestALevelUpIsDrawnAmongTheKindsItsLevelUnlockedWhileTheGatesAreOn(t *testing.T) {
	rules := TestGated()
	rules.Kinds = map[Kind]float64{KindBomb: 1, KindEncloseClicks: 1}

	for range 30 {
		reward, ok := LevelUp{Unlocked: Unlocked{KindEncloseClicks}}.Reward(rules, nil)
		require.True(t, ok)
		assert.Equal(t, KindEncloseClicks, reward.Kind, "level 4 unlocks no bomb")
	}
}

func TestALevelUpIsPaidInABoxsAmounts(t *testing.T) {
	rules := TestRules()
	rules.Kinds = map[Kind]float64{KindShields: 1}

	reward, ok := LevelUp{}.Reward(rules, nil)

	require.True(t, ok)
	assert.GreaterOrEqual(t, reward.Amount, rules.Shield.MinPerBox)
	assert.LessOrEqual(t, reward.Amount, rules.Shield.MaxPerBox)
}

func TestAGrantIdThatIsNoIdIsRefused(t *testing.T) {
	for _, bad := range []string{"", "a-token", "00000000-0000-0000-0000-000000000000"} {
		_, err := GrantIDOf(bad)
		require.ErrorIs(t, err, ErrInvalidGrant, "%q", bad)
	}
}
