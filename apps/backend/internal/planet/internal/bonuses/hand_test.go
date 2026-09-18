package bonuses

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var chargeRules = ChargesConfig{TTL: 24 * time.Hour, SpreadClicks: 8, EnclosureMaxTiles: 25}

func TestAHolderIsTheAccount(t *testing.T) {
	assert.Equal(t, Holder("acc-1"), HolderOf(clicks.Payer{Scope: "1.2.3.4", Account: "acc-1"}))
	assert.Equal(t, NoHolder, HolderOf(clicks.Payer{Scope: "1.2.3.4"}), "an address alone holds nothing")
}

func TestABombIsKeptUntilItIsDropped(t *testing.T) {
	hand := Hand{}.Granted(KindBomb, epoch, chargeRules)

	assert.Equal(t, Held{Bomb: true}, hand.Held(epoch.Add(23*time.Hour)), "no timer: a bomb waits for its moment")

	dropped, ok := hand.AfterBomb(epoch.Add(23 * time.Hour))
	require.True(t, ok)
	assert.Equal(t, Held{}, dropped.Held(epoch))

	_, again := dropped.AfterBomb(epoch)
	assert.False(t, again, "a bomb that went off is gone")
}

func TestAChargeHeldPastItsExpiryIsLost(t *testing.T) {
	hand := Hand{}.
		Granted(KindBomb, epoch, chargeRules).
		Granted(KindEncloseClicks, epoch, chargeRules).
		Granted(KindSpreadClicks, epoch, chargeRules)
	later := epoch.Add(24 * time.Hour)

	assert.True(t, hand.Empty(later))

	_, bomb := hand.AfterBomb(later)
	_, enclose := hand.AfterEnclose(later)
	_, spread := hand.AfterSpreadClick(later)
	assert.False(t, bomb || enclose || spread)
}

func TestNobodyHoldsTwoOfOneKind(t *testing.T) {
	hand := Hand{}.Granted(KindBomb, epoch, chargeRules).Granted(KindBomb, epoch, chargeRules)

	dropped, ok := hand.AfterBomb(epoch)
	require.True(t, ok)
	_, again := dropped.AfterBomb(epoch)
	assert.False(t, again, "a second grant replaces the first rather than stacking on it")
}

func TestAnEncloseChargeIsOneShape(t *testing.T) {
	hand := Hand{}.Granted(KindEncloseClicks, epoch, chargeRules)

	closed, ok := hand.AfterEnclose(epoch)
	require.True(t, ok)
	_, again := closed.AfterEnclose(epoch)
	assert.False(t, again)
}

func TestASpreadChargeIsSpentOneClickAtATime(t *testing.T) {
	hand := Hand{}.Granted(KindSpreadClicks, epoch, chargeRules)

	for left := 7; left >= 0; left-- {
		var ok bool
		hand, ok = hand.AfterSpreadClick(epoch)
		require.True(t, ok)
		require.Equal(t, left, hand.Held(epoch).SpreadClicks)
	}

	_, ok := hand.AfterSpreadClick(epoch)
	assert.False(t, ok, "eight clicks, and no ninth")
	assert.True(t, hand.Empty(epoch))
}

func TestASecondSpreadRefillsTheClicksRatherThanAddingToThem(t *testing.T) {
	hand, _ := Hand{}.Granted(KindSpreadClicks, epoch, chargeRules).AfterSpreadClick(epoch)

	assert.Equal(t, 8, hand.Granted(KindSpreadClicks, epoch, chargeRules).Held(epoch).SpreadClicks)
}

func TestATripleIsNotACharge(t *testing.T) {
	assert.True(t, Hand{}.Granted(KindTripleClicks, epoch, chargeRules).Empty(epoch))
}

func TestAHandIsAValue(t *testing.T) {
	hand := Hand{}.Granted(KindBomb, epoch, chargeRules)

	_, _ = hand.AfterBomb(epoch)

	assert.True(t, hand.Held(epoch).Bomb, "spending builds a new hand and leaves this one as it was")
}

func TestHeldNamesEveryKindInHand(t *testing.T) {
	assert.Empty(t, Held{}.Kinds())
	assert.ElementsMatch(t, []Kind{KindBomb, KindEncloseClicks, KindSpreadClicks},
		Held{Bomb: true, Enclose: true, SpreadClicks: 3}.Kinds())
}
