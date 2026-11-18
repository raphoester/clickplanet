package charges

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var chargeRules = func() Rules {
	rules := TestRules()
	rules.Shield.Held = 12
	return rules
}()

func TestABombIsKeptUntilItIsDropped(t *testing.T) {
	held := Held{}.Granted(KindBomb, 1, chargeRules)
	require.Equal(t, Held{Bombs: 1}, held, "nothing lapses: a bomb waits for its moment")

	dropped, ok := held.AfterBomb()
	require.True(t, ok)
	assert.True(t, dropped.Empty())

	_, again := dropped.AfterBomb()
	assert.False(t, again, "a bomb that went off is gone")
}

func TestBombsAndRefillsStackToWhatAHandHolds(t *testing.T) {
	held := Held{}.Granted(KindBomb, 1, chargeRules).Granted(KindBomb, 1, chargeRules).
		Granted(KindRefill, 1, chargeRules).Granted(KindRefill, 1, chargeRules)
	require.Equal(t, Held{Bombs: 1, Refills: 1}, held, "a hand holds one of each while the rules say one")

	more := chargeRules
	more.Bomb.Held, more.Refill.Held = 3, 2
	held = Held{}.Granted(KindBomb, 2, more).Granted(KindBomb, 2, more).
		Granted(KindRefill, 1, more).Granted(KindRefill, 1, more).Granted(KindRefill, 1, more)
	require.Equal(t, Held{Bombs: 3, Refills: 2}, held)

	held, bomb := held.AfterBomb()
	held, refill := held.AfterRefill()
	require.True(t, bomb && refill)
	assert.Equal(t, Held{Bombs: 2, Refills: 1}, held, "each is spent alone")
}

func TestEveryKindStacksToItsSize(t *testing.T) {
	for _, kind := range Kinds {
		held := Held{}.Granted(kind, 1000, chargeRules)

		assert.Equal(t, chargeRules.Size(kind), held.Count(kind), kind)
		assert.Equal(t, held, held.Granted(kind, 1, chargeRules), "%s: a full stack takes no more", kind)
	}
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

func TestShieldsStackUpToTheirSize(t *testing.T) {
	held := Held{Shields: 10}.Granted(KindShields, 3, chargeRules)

	assert.Equal(t, 12, held.Shields, "twelve at most, however many the box gave")
}

func TestAStackHeldAboveASmallerSizeIsKeptAndTakesNothingMore(t *testing.T) {
	smaller := chargeRules
	smaller.Shield.Held = 5

	held := Held{Shields: 12}.Granted(KindShields, 3, smaller)

	assert.Equal(t, 12, held.Shields)
	assert.Equal(t, held, held.Granted(KindShields, 1, smaller), "a grant adds none to a stack above its size")
}

func TestEachShieldIsPlacedOnce(t *testing.T) {
	held := Held{Shields: 2}

	held, first := held.AfterShield()
	held, second := held.AfterShield()
	_, third := held.AfterShield()

	assert.True(t, first && second)
	assert.False(t, third)
	assert.True(t, held.Empty())
}

func TestAShieldGivenBackIsTheOneTakenWhateverTheStackSize(t *testing.T) {
	smaller := chargeRules
	smaller.Shield.Held = 5

	spent, ok := Held{Shields: 12}.AfterShield()
	require.True(t, ok)

	assert.Equal(t, Held{Shields: 12}, spent.WithShieldBack())
	assert.Equal(t, 11, spent.Granted(KindShields, 1, smaller).Shields, "a grant would have kept it")
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

	assert.Equal(t, 1, held.Bombs, "spending builds a new value and leaves this one as it was")
}

func TestCountIsHowManyOfAKindAreHeld(t *testing.T) {
	held := Held{Bombs: 1, Enclosures: 2, SpreadClicks: 5, Shields: 7, Viruses: 1}

	assert.Equal(t, 7, held.Count(KindShields))
	assert.Equal(t, 0, held.Count(KindRefill))
	assert.Equal(t, 1, held.Count(KindBomb))
	assert.Equal(t, 2, held.Count(KindEncloseClicks))
	assert.Equal(t, 5, held.Count(KindSpreadClicks))
	assert.Equal(t, 1, held.Count(KindVirus))
}

func TestOnlyAnAccountIDIsAHolder(t *testing.T) {
	for _, account := range []string{"", "a-guest", "1.2.3.4", "00000000-0000-0000-0000-000000000000"} {
		_, err := ParseHolder(account)
		require.ErrorIs(t, err, ErrInvalidHolder, account)
	}
}

func TestAGrantNamesAtLeastOneCharge(t *testing.T) {
	require.ErrorIs(t, Held{}.GrantError(), ErrNothingToGrant)

	for _, grant := range []Held{
		{Refills: 1}, {Bombs: 1}, {Enclosures: 1}, {SpreadClicks: 1}, {Shields: 1}, {Viruses: 1}, {Mines: 1}, {Rallies: 1},
	} {
		assert.NoError(t, grant.GrantError(), grant)
	}
}

func TestAGrantTakesNothingAway(t *testing.T) {
	require.ErrorIs(t, Held{Bombs: 1, Enclosures: -1}.GrantError(), ErrNegativeGrant)
	require.ErrorIs(t, Held{Refills: -1}.GrantError(), ErrNegativeGrant)
	require.ErrorIs(t, Held{Bombs: -1}.GrantError(), ErrNegativeGrant)
	require.ErrorIs(t, Held{Viruses: -1}.GrantError(), ErrNegativeGrant)
	require.ErrorIs(t, Held{SpreadClicks: -1}.GrantError(), ErrNegativeGrant)
	require.ErrorIs(t, Held{Shields: -1}.GrantError(), ErrNegativeGrant)
}

func TestAGrantKeepsNoBombWhileBombsAreOff(t *testing.T) {
	off := TestRules()
	off.Bomb.Held = 3
	off.Bomb.Off = true

	assert.Equal(t, Held{}, Held{}.Granted(KindBomb, 1, off), "an empty hand takes no bomb")
	assert.Equal(t, Held{Bombs: 1}, Held{Bombs: 1}.Granted(KindBomb, 1, off), "and the bombs held are kept")
	assert.Equal(t, Held{Shields: 5}, Held{}.Granted(KindShields, 5, off), "the other kinds are granted")
}

func TestAGainTopsAStackUpToItsSize(t *testing.T) {
	gain := Held{SpreadClicks: 6}.Gain(KindSpreadClicks, 4, chargeRules)

	assert.Equal(t, []Kind{KindSpreadClicks}, gain.Paid)
	assert.Equal(t, 2, gain.Count(KindSpreadClicks))
	assert.Equal(t, Held{SpreadClicks: 8}, gain.After)
}

func TestAGainIntoAFullStackPaysTheKindAndKeepsNone(t *testing.T) {
	gain := Held{Bombs: 1, Shields: 3}.Gain(KindBomb, 1, chargeRules)

	assert.Equal(t, []Kind{KindBomb}, gain.Paid)
	assert.Zero(t, gain.Count(KindBomb))
	assert.Equal(t, Held{Bombs: 1, Shields: 3}, gain.After)
}

func TestAGainPaysNoKindTheRulesTurnOff(t *testing.T) {
	off := TestRules()
	off.Bomb.Off = true

	gain := Held{}.Gain(KindBomb, 1, off)

	assert.Empty(t, gain.Paid)
	assert.Equal(t, Held{}, gain.After)
}

func TestAFillingTopsEachStackNamedUpToItsSize(t *testing.T) {
	gain := Held{Shields: 5, SpreadClicks: 8, Mines: 1}.Filling([]Kind{KindShields, KindSpreadClicks, KindBomb}, chargeRules)

	assert.Equal(t, []Kind{KindSpreadClicks, KindBomb, KindShields}, gain.Paid, "a full stack is paid too")
	assert.Equal(t, Held{Shields: 12, SpreadClicks: 8, Bombs: 1, Mines: 1}, gain.After, "a kind not named is left")
	assert.Equal(t, 7, gain.Count(KindShields))
	assert.Zero(t, gain.Count(KindSpreadClicks))
}

func TestAFillingNeverTakesFromAStackHeldAboveItsSize(t *testing.T) {
	gain := Held{Shields: 20}.Filling([]Kind{KindShields}, chargeRules)

	assert.Equal(t, Held{Shields: 20}, gain.After)
}

func TestAFillingPaysNoKindTheRulesTurnOff(t *testing.T) {
	off := TestRules()
	off.Bomb.Off = true

	gain := Held{}.Filling([]Kind{KindBomb, KindRefill}, off)

	assert.Equal(t, []Kind{KindRefill}, gain.Paid)
	assert.Equal(t, Held{Refills: 1}, gain.After)
}

func TestAGrantKeepsNoEnclosureWhileEnclosuresAreOff(t *testing.T) {
	off := TestRules()
	off.Enclose.Off = true

	assert.Equal(t, Held{}, Held{}.Granted(KindEncloseClicks, 2, off), "an empty hand takes no enclosure")
	assert.Equal(t, Held{Enclosures: 1}, Held{Enclosures: 1}.Granted(KindEncloseClicks, 1, off))
}

func TestAGrantKeepsNoSpreadClickWhileSpreadsAreOff(t *testing.T) {
	off := TestRules()
	off.Spread.Off = true

	assert.Equal(t, Held{}, Held{}.Granted(KindSpreadClicks, 4, off), "an empty hand takes no spread click")
	assert.Equal(t, Held{SpreadClicks: 2}, Held{SpreadClicks: 2}.Granted(KindSpreadClicks, 4, off), "and one held keeps its own")
	assert.Empty(t, Held{}.Gain(KindSpreadClicks, 4, off).Paid, "a spread is not paid")
	assert.Equal(t, []Kind{KindShields}, Held{}.Filling([]Kind{KindSpreadClicks, KindShields}, off).Paid)
}

func TestAVirusIsKeptUntilItIsUsedAndAHandHoldsWhatTheRulesSay(t *testing.T) {
	held := Held{}.Granted(KindVirus, 1, chargeRules)
	require.Equal(t, Held{Viruses: 1}, held)
	assert.Equal(t, held, held.Granted(KindVirus, 1, chargeRules), "a hand of one takes no second one")

	used, ok := held.AfterVirus()
	require.True(t, ok)
	assert.True(t, used.Empty())
	_, again := used.AfterVirus()
	assert.False(t, again)
	assert.Equal(t, held, used.WithVirusBack(), "a virus the map refused comes back")
}

func TestMinesStackToWhatAHandHoldsAndEachIsSpentAlone(t *testing.T) {
	held := Held{}.Granted(KindMine, 2, chargeRules).Granted(KindMine, 2, chargeRules)
	require.Equal(t, Held{Mines: 3}, held, "a hand holds 3")
	assert.Equal(t, 3, held.Count(KindMine))

	used, ok := held.AfterMine()
	require.True(t, ok)
	assert.Equal(t, Held{Mines: 2}, used)
	_, none := Held{}.AfterMine()
	assert.False(t, none)
	assert.Equal(t, held, used.WithMineBack(), "a mine the map refused comes back")
	assert.ErrorIs(t, Held{Mines: -1}.GrantError(), ErrNegativeGrant)
}

func TestRalliesStackToWhatAHandHoldsAndEachIsSpentAlone(t *testing.T) {
	held := Held{}.Granted(KindRally, 2, chargeRules).Granted(KindRally, 2, chargeRules)
	require.Equal(t, Held{Rallies: 3}, held, "a hand holds 3")
	assert.Equal(t, 3, held.Count(KindRally))
	assert.Equal(t, Held{Rallies: 2}, GrantOf(KindRally, 2))

	used, ok := held.AfterRally()
	require.True(t, ok)
	assert.Equal(t, Held{Rallies: 2}, used)
	_, none := Held{}.AfterRally()
	assert.False(t, none)
	assert.Equal(t, held, used.WithRallyBack(), "a rally refused once spent comes back")
	assert.ErrorIs(t, Held{Rallies: -1}.GrantError(), ErrNegativeGrant)
}
