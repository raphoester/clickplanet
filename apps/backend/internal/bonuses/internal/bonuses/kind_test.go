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
	seen := cpcolls.NewSet[Kind]()
	for range 200 {
		seen.Add(drawKind(everyKind(), TestRules().Kinds))
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
	assert.Equal(t, everyKind(), TestGated().drawable(nil, Unlocked{KindEncloseClicks, KindBomb}))
}

func TestWithTheGatesOffEveryKindIsDrawnToAGuest(t *testing.T) {
	assert.Equal(t, everyKind(), TestRules().drawable(nil, nil), "a battle's set keeps the gates off")
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

func TestAWelcomeOfAKindNotOpenIsWorthNothing(t *testing.T) {
	rules := TestGated()
	rules.Gates.Open[KindSpreadClicks] = false

	_, ok := CaughtBox{Welcome: true}.Reward(rules, nil, nil)

	assert.False(t, ok)
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
