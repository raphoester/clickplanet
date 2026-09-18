package bonuses

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	alice Holder = "account:alice"
	bob   Holder = "account:bob"
)

func newTestCharges() (*Charges, *cptime.FixedClock) {
	clock := cptime.NewFixedClock(epoch)

	return NewCharges(ChargesConfig{TTL: 24 * time.Hour, SpreadClicks: 8, EnclosureMaxTiles: 25}, clock), clock
}

func TestAHolderIsTheAccountWhenThereIsOneAndTheScopeOtherwise(t *testing.T) {
	assert.Equal(t, Holder("account:acc-1"), HolderOf(clicks.Payer{Scope: "1.2.3.4", Account: "acc-1"}))
	assert.Equal(t, Holder("scope:1.2.3.4"), HolderOf(clicks.Payer{Scope: "1.2.3.4"}))
	assert.NotEqual(t, HolderOf(clicks.Payer{Scope: "x"}), HolderOf(clicks.Payer{Scope: "other", Account: "x"}),
		"an account never shares a holder with a scope that happens to read the same")
}

func TestABombIsKeptUntilItIsDropped(t *testing.T) {
	charges, clock := newTestCharges()
	charges.Grant(alice, KindBomb)

	clock.Advance(23 * time.Hour)
	assert.Equal(t, Held{Bomb: true}, charges.Held(alice), "no timer: a bomb waits for its moment")

	assert.True(t, charges.SpendBomb(alice))
	assert.False(t, charges.SpendBomb(alice), "a bomb that went off is gone")
	assert.Equal(t, Held{}, charges.Held(alice))
}

func TestAChargeHeldPastItsExpiryIsLost(t *testing.T) {
	charges, clock := newTestCharges()
	charges.Grant(alice, KindBomb)
	charges.Grant(alice, KindEncloseClicks)
	charges.Grant(alice, KindSpreadClicks)

	clock.Advance(24 * time.Hour)

	assert.Equal(t, Held{}, charges.Held(alice))
	assert.False(t, charges.SpendBomb(alice))
	assert.False(t, charges.SpendEnclose(alice))
	assert.False(t, charges.SpendSpreadClick(alice))
}

func TestAChargeBelongsToItsHolder(t *testing.T) {
	charges, _ := newTestCharges()
	charges.Grant(alice, KindBomb)

	assert.False(t, charges.SpendBomb(bob))
	assert.Equal(t, Held{}, charges.Held(bob))
	assert.True(t, charges.SpendBomb(alice))
}

func TestNobodyHoldsTwoOfOneKind(t *testing.T) {
	charges, _ := newTestCharges()
	charges.Grant(alice, KindBomb)
	charges.Grant(alice, KindBomb)

	assert.True(t, charges.SpendBomb(alice))
	assert.False(t, charges.SpendBomb(alice), "a second grant replaces the first rather than stacking on it")
}

func TestAnEncloseChargeIsOneShape(t *testing.T) {
	charges, _ := newTestCharges()
	charges.Grant(alice, KindEncloseClicks)

	assert.Equal(t, 25, charges.EnclosureMaxTiles())
	assert.Equal(t, 8, charges.SpreadClicks())
	assert.True(t, charges.SpendEnclose(alice))
	assert.False(t, charges.SpendEnclose(alice))
}

func TestASpreadChargeIsSpentOneClickAtATime(t *testing.T) {
	charges, _ := newTestCharges()
	charges.Grant(alice, KindSpreadClicks)

	for left := 7; left >= 0; left-- {
		require.True(t, charges.SpendSpreadClick(alice))
		require.Equal(t, left, charges.Held(alice).SpreadClicks)
	}

	assert.False(t, charges.SpendSpreadClick(alice), "eight clicks, and no ninth")
	assert.Empty(t, charges.Held(alice).Kinds())
}

func TestASecondSpreadRefillsTheClicksRatherThanAddingToThem(t *testing.T) {
	charges, _ := newTestCharges()
	charges.Grant(alice, KindSpreadClicks)
	require.True(t, charges.SpendSpreadClick(alice))

	charges.Grant(alice, KindSpreadClicks)

	assert.Equal(t, 8, charges.Held(alice).SpreadClicks)
}

func TestATripleIsNotACharge(t *testing.T) {
	charges, _ := newTestCharges()
	charges.Grant(alice, KindTripleClicks)

	assert.Equal(t, Held{}, charges.Held(alice))
}

func TestAGrantForgetsTheHandsWithNothingLeft(t *testing.T) {
	charges, clock := newTestCharges()
	charges.Grant(alice, KindBomb)
	charges.Grant(bob, KindBomb)
	require.True(t, charges.SpendBomb(bob))

	clock.Advance(25 * time.Hour)
	charges.Grant("account:carol", KindBomb)

	assert.Len(t, charges.held, 1, "a spent or lapsed hand is not kept forever")
}

func TestHeldNamesEveryKindInHand(t *testing.T) {
	assert.Empty(t, Held{}.Kinds())
	assert.ElementsMatch(t, []Kind{KindBomb, KindEncloseClicks, KindSpreadClicks},
		Held{Bomb: true, Enclose: true, SpreadClicks: 3}.Kinds())
}
