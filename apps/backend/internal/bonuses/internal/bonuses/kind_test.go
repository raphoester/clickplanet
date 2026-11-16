package bonuses

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

func everyKind() *cpcolls.Set[Kind] {
	return cpcolls.NewSet(Kinds...)
}

// testWeighed is every kind TestRules weighs above 0: all but the virus, the mine and the rally.
func testWeighed() *cpcolls.Set[Kind] {
	return cpcolls.NewSet(KindRefill, KindSpreadClicks, KindEncloseClicks, KindBomb, KindShields)
}

func TestEveryKindWeighedIsDrawn(t *testing.T) {
	weights := TestRules().Kinds
	weights[KindVirus] = 1
	weights[KindMine] = 1
	weights[KindRally] = 1
	seen := cpcolls.NewSet[Kind]()
	for range 500 {
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

		assert.Equal(t, cpcolls.NewSet(KindSpreadClicks), rules.drawable(nil, nil))
	}
}

func TestAKindAHandHasNoRoomForIsLeftOut(t *testing.T) {
	assert.Equal(t, cpcolls.NewSet(KindSpreadClicks, KindEncloseClicks, KindShields),
		TestRules().drawable([]Kind{KindRefill, KindBomb}, nil))
}

func TestWithTheGatesOnAKindIsDrawnOnlyOpenOrUnlocked(t *testing.T) {
	open := []Kind{KindRefill, KindSpreadClicks, KindShields}

	assert.Equal(t, cpcolls.NewSet(open...), TestGated().drawable(nil, nil), "a guest has nothing unlocked")
	assert.Equal(t, cpcolls.NewSet(append(open[:3:3], KindEncloseClicks)...),
		TestGated().drawable(nil, Unlocked{KindEncloseClicks}))
	assert.Equal(t, testWeighed(), TestGated().drawable(nil, Unlocked{KindEncloseClicks, KindBomb}))
}

func TestAVirusWeighedIsDrawnWithTheGatesOnOnlyOnceUnlocked(t *testing.T) {
	rules := TestGated()
	rules.Kinds[KindVirus] = 1

	assert.False(t, rules.drawable(nil, Unlocked{KindEncloseClicks, KindBomb}).Contains(KindVirus))
	assert.True(t, rules.drawable(nil, Unlocked{KindVirus}).Contains(KindVirus))
}

func TestAMineWeighedIsDrawnWithTheGatesOnOnlyOnceUnlocked(t *testing.T) {
	rules := TestGated()
	rules.Kinds[KindMine] = 1

	assert.False(t, rules.drawable(nil, Unlocked{KindEncloseClicks, KindBomb, KindVirus}).Contains(KindMine))
	assert.True(t, rules.drawable(nil, Unlocked{KindMine}).Contains(KindMine))
	assert.False(t, rules.drawable([]Kind{KindMine}, Unlocked{KindMine}).Contains(KindMine), "a hand of 3 has no room")
}

func TestARallyWeighedIsDrawnWithTheGatesOnOnlyOnceUnlocked(t *testing.T) {
	rules := TestGated()
	rules.Kinds[KindRally] = 1

	assert.False(t, rules.drawable(nil, Unlocked{KindEncloseClicks, KindBomb, KindVirus, KindMine}).Contains(KindRally))
	assert.True(t, rules.drawable(nil, Unlocked{KindRally}).Contains(KindRally))
	assert.False(t, rules.drawable([]Kind{KindRally}, Unlocked{KindRally}).Contains(KindRally), "a hand of 3 has no room")
}

func TestWithTheGatesOffEveryKindIsDrawnToAGuest(t *testing.T) {
	assert.Equal(t, testWeighed(), TestRules().drawable(nil, nil), "a battle's set keeps the gates off")
}

func TestAWinWhoseKindsAllWeighZeroIsWorthNothing(t *testing.T) {
	rules := TestGated()
	rules.Kinds = map[Kind]float64{KindBomb: 1, KindEncloseClicks: 1}

	_, ok := CaughtBox{}.Reward(rules, nil, Unlocked{KindEncloseClicks})
	assert.True(t, ok)

	_, ok = CaughtBox{}.Reward(rules, nil, nil)
	assert.False(t, ok, "the box pays nothing, as for a full hand: never a kind the account may not be drawn")
	_, ok = Answer{}.Reward(rules, nil, nil)
	assert.False(t, ok)
}

func TestAGuestsWelcomeIsItsKindThoughTheGatesCloseIt(t *testing.T) {
	rules := TestGated()
	rules.Gates.Open[KindSpreadClicks] = false
	rules.Gates.Open[KindShields] = false

	box, ok := CaughtBox{Welcome: true}.Reward(rules, nil, nil)
	require.True(t, ok)
	assert.Equal(t, KindSpreadClicks, box.Kind)

	answer, ok := Answer{Welcome: true}.Reward(rules, nil, nil)
	require.True(t, ok)
	assert.Equal(t, KindShields, answer.Kind)

	_, ok = CaughtBox{}.Reward(rules, []Kind{KindRefill}, nil)
	assert.False(t, ok, "past the welcome, a guest holding a refill is drawn nothing")
}

func TestAWelcomeOfAKindWeighed0OrWithNoRoomIsWorthNothing(t *testing.T) {
	rules := TestGated()
	rules.Kinds[KindShields] = 0

	_, ok := Answer{Welcome: true}.Reward(rules, nil, nil)
	assert.False(t, ok, "weighed 0")
	_, ok = CaughtBox{Welcome: true}.Reward(rules, []Kind{KindSpreadClicks}, nil)
	assert.False(t, ok, "no room")
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
}

func TestAWelcomeBoxIsASpreadAndAWelcomeAnswerShields(t *testing.T) {
	for range 50 {
		box, ok := CaughtBox{Welcome: true}.Reward(TestRules(), nil, nil)
		assert.True(t, ok)
		assert.Equal(t, KindSpreadClicks, box.Kind)

		answer, ok := Answer{Welcome: true}.Reward(TestRules(), nil, nil)
		assert.True(t, ok)
		assert.Equal(t, KindShields, answer.Kind)
	}
}
