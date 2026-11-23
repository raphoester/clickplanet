package bonuses

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

func everyKind() *cpcolls.Set[Kind] {
	return cpcolls.NewSet(Kinds...)
}

func TestEveryKindWeighedIsDrawn(t *testing.T) {
	weights := TestRules().Kinds
	weights[KindVirus] = 1
	weights[KindMine] = 1
	weights[KindRally] = 1
	weights[KindTrap] = 1
	seen := cpcolls.NewSet[Kind]()
	for range 600 {
		seen.Add(drawKind(everyKind(), weights))
	}

	assert.Equal(t, len(Kinds), seen.Len())
}

func TestKindsAreDrawnInProportionToTheirWeight(t *testing.T) {
	weights := map[Kind]float64{KindRefill: 9, KindSpreadClicks: 1}

	const draws = 20_000
	spreads := 0
	for range draws {
		if drawKind(everyKind(), weights) == KindSpreadClicks {
			spreads++
		}
	}

	assert.InDelta(t, 0.1, float64(spreads)/draws, 0.02)
}

func TestAKindLeftOutOrAtZeroIsNeverDrawn(t *testing.T) {
	for _, kinds := range []map[Kind]float64{
		{KindSpreadClicks: 1},
		{KindSpreadClicks: 1, KindRefill: 0},
	} {
		rules := TestRules()
		rules.Kinds = kinds

		assert.Equal(t, cpcolls.NewSet(KindSpreadClicks), rules.drawable(nil))
	}
}

func TestAVirusWeighedIsDrawnWithTheGatesOnOnlyOnceUnlocked(t *testing.T) {
	rules := TestGated()
	rules.Kinds[KindVirus] = 1

	assert.False(t, rules.drawable(Unlocked{KindEncloseClicks, KindBomb}).Contains(KindVirus))
	assert.True(t, rules.drawable(Unlocked{KindVirus}).Contains(KindVirus))
}

func TestAMineWeighedIsDrawnWithTheGatesOnOnlyOnceUnlocked(t *testing.T) {
	rules := TestGated()
	rules.Kinds[KindMine] = 1

	assert.False(t, rules.drawable(Unlocked{KindEncloseClicks, KindBomb, KindVirus}).Contains(KindMine))
	assert.True(t, rules.drawable(Unlocked{KindMine}).Contains(KindMine))
}

func TestATrapWeighedIsDrawnWithTheGatesOnOnlyOnceUnlocked(t *testing.T) {
	rules := TestGated()
	rules.Kinds[KindTrap] = 1

	assert.False(t, rules.drawable(Unlocked{KindEncloseClicks, KindBomb, KindVirus, KindMine, KindRally}).Contains(KindTrap))
	assert.True(t, rules.drawable(Unlocked{KindTrap}).Contains(KindTrap))
}

func TestARallyWeighedIsDrawnWithTheGatesOnOnlyOnceUnlocked(t *testing.T) {
	rules := TestGated()
	rules.Kinds[KindRally] = 1

	assert.False(t, rules.drawable(Unlocked{KindEncloseClicks, KindBomb, KindVirus, KindMine}).Contains(KindRally))
	assert.True(t, rules.drawable(Unlocked{KindRally}).Contains(KindRally))
}

func TestABoxGivesAFewEnclosuresSpreadClicksOrShieldsAndOneOfTheRest(t *testing.T) {
	seen := map[Kind]*cpcolls.Set[int]{}
	for _, kind := range Kinds {
		seen[kind] = cpcolls.NewSet[int]()
		for range 1000 {
			seen[kind].Add(amountOf(kind, TestRules()))
		}
	}

	assert.Equal(t, cpcolls.NewSet(1, 2, 3, 4), seen[KindSpreadClicks], "1 to 4 spread clicks")
	assert.Equal(t, cpcolls.NewSet(1, 2, 3), seen[KindEncloseClicks], "1 to 3 enclosures")
	assert.Equal(t, cpcolls.NewSet(5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20), seen[KindShields], "5 to 20 shields")
	assert.Equal(t, cpcolls.NewSet(1), seen[KindBomb])
	assert.Equal(t, cpcolls.NewSet(1), seen[KindRefill])
	assert.Equal(t, cpcolls.NewSet(1, 2), seen[KindMine], "1 or 2 mines")
	assert.Equal(t, cpcolls.NewSet(1), seen[KindRally], "one rally")
	assert.Equal(t, cpcolls.NewSet(1, 2), seen[KindTrap], "1 or 2 traps")
}
