package bonuses

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var chargeRules = ChargesConfig{SpreadClicks: 8, Enclosures: 3, EnclosureMaxTiles: 25}

func TestAHolderIsTheAccount(t *testing.T) {
	assert.Equal(t, Holder("acc-1"), HolderOf(clicks.Payer{Scope: "1.2.3.4", Account: "acc-1"}))
	assert.Equal(t, NoHolder, HolderOf(clicks.Payer{Scope: "1.2.3.4"}), "an address alone holds nothing")
}

func TestABombIsKeptUntilItIsDropped(t *testing.T) {
	held := Held{}.Granted(KindBomb, 1, chargeRules)
	require.Equal(t, Held{Bomb: true}, held, "nothing lapses: a bomb waits for its moment")

	dropped, ok := held.AfterBomb()
	require.True(t, ok)
	assert.True(t, dropped.Empty())

	_, again := dropped.AfterBomb()
	assert.False(t, again, "a bomb that went off is gone")
}

func TestNobodyHoldsTwoBombsOrTwoRefills(t *testing.T) {
	held := Held{}.Granted(KindBomb, 1, chargeRules).Granted(KindBomb, 1, chargeRules).
		Granted(KindRefill, 1, chargeRules).Granted(KindRefill, 1, chargeRules)

	held, bomb := held.AfterBomb()
	held, refill := held.AfterRefill()
	require.True(t, bomb && refill)

	_, bomb = held.AfterBomb()
	_, refill = held.AfterRefill()
	assert.False(t, bomb || refill, "a second grant replaces the first rather than stacking on it")
}

func TestEnclosuresStackUpToTheirSize(t *testing.T) {
	held := Held{}.Granted(KindEncloseClicks, 2, chargeRules)
	require.Equal(t, 2, held.Enclosures)

	held = held.Granted(KindEncloseClicks, 3, chargeRules)
	assert.Equal(t, 3, held.Enclosures, "three at most, however many the box gave")
}

func TestEachEnclosureIsOneShape(t *testing.T) {
	held := Held{}.Granted(KindEncloseClicks, 2, chargeRules)

	held, first := held.AfterEnclose()
	held, second := held.AfterEnclose()
	_, third := held.AfterEnclose()

	assert.True(t, first && second)
	assert.False(t, third)
}

func TestSpreadClicksAddUpToThePoolsSize(t *testing.T) {
	held := Held{SpreadClicks: 3}.Granted(KindSpreadClicks, 4, chargeRules)
	require.Equal(t, 7, held.SpreadClicks)

	held = held.Granted(KindSpreadClicks, 4, chargeRules)
	assert.Equal(t, 8, held.SpreadClicks, "eight at most")
}

func TestTheSpreadPoolIsSpentOneClickAtATime(t *testing.T) {
	held := Held{SpreadClicks: 2}

	held, first := held.AfterSpreadClick()
	held, second := held.AfterSpreadClick()
	_, third := held.AfterSpreadClick()

	assert.True(t, first && second)
	assert.False(t, third)
	assert.True(t, held.Empty())
}

func TestARefillIsOneFill(t *testing.T) {
	used, ok := Held{}.Granted(KindRefill, 1, chargeRules).AfterRefill()
	require.True(t, ok)

	_, again := used.AfterRefill()
	assert.False(t, again)
}

func TestAHeldIsAValue(t *testing.T) {
	held := Held{}.Granted(KindBomb, 1, chargeRules)

	_, _ = held.AfterBomb()

	assert.True(t, held.Bomb, "spending builds a new value and leaves this one as it was")
}

func TestFullNamesTheKindsAnotherBoxWouldAddNothingTo(t *testing.T) {
	assert.Empty(t, Held{SpreadClicks: 7, Enclosures: 2}.Full(chargeRules), "a pool or a stack with room takes another box")
	assert.ElementsMatch(t, []Kind{KindRefill, KindBomb, KindEncloseClicks, KindSpreadClicks},
		Held{Refill: true, Bomb: true, Enclosures: 3, SpreadClicks: 8}.Full(chargeRules))
}

func TestCountIsHowManyOfAKindAreHeld(t *testing.T) {
	held := Held{Bomb: true, Enclosures: 2, SpreadClicks: 5}

	assert.Equal(t, 0, held.Count(KindRefill))
	assert.Equal(t, 1, held.Count(KindBomb))
	assert.Equal(t, 2, held.Count(KindEncloseClicks))
	assert.Equal(t, 5, held.Count(KindSpreadClicks))
}
