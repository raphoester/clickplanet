package charges_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
)

func spreadRules(rings int, unlimited bool) func() charges.Rules {
	return func() charges.Rules {
		rules := charges.TestRules()
		rules.Spread.Rings, rules.Spread.Unlimited = rings, unlimited
		return rules
	}
}

func TestASpreadTakesOneSpreadClickAndReachesTheRingsInForce(t *testing.T) {
	store := charges.NewMemoryStore()
	store.Keep("alice", charges.Held{SpreadClicks: 2})
	wallet := charges.NewCharges(store, spreadRules(2, false))

	spread, err := wallet.TakeSpread(t.Context(), "alice")

	require.NoError(t, err)
	assert.Equal(t, charges.Spread{Rings: 2, Spent: true}, spread)
	assert.Equal(t, 1, store.Kept("alice").SpreadClicks)
}

func TestWithNoSpreadClickHeldAClickSpreadsNothing(t *testing.T) {
	wallet := charges.NewCharges(charges.NewMemoryStore(), spreadRules(2, false))

	spread, err := wallet.TakeSpread(t.Context(), "alice")
	require.NoError(t, err)
	assert.Equal(t, charges.Spread{}, spread)

	guest, err := wallet.TakeSpread(t.Context(), charges.NoHolder)
	require.NoError(t, err)
	assert.Equal(t, charges.Spread{}, guest)
}

func TestAnUnlimitedSpreadSpendsNothingAndNeedsNothing(t *testing.T) {
	store := charges.NewMemoryStore()
	store.Keep("alice", charges.Held{SpreadClicks: 3})
	wallet := charges.NewCharges(store, spreadRules(2, true))

	held, err := wallet.TakeSpread(t.Context(), "alice")
	require.NoError(t, err)
	guest, err := wallet.TakeSpread(t.Context(), charges.NoHolder)
	require.NoError(t, err)

	assert.Equal(t, charges.Spread{Rings: 2}, held)
	assert.Equal(t, charges.Spread{Rings: 2}, guest)
	assert.Equal(t, 3, store.Kept("alice").SpreadClicks, "the pool held is left for when the set stops")
}

func TestASpreadGivenBackIsTheSpreadClickItSpent(t *testing.T) {
	store := charges.NewMemoryStore()
	store.Keep("alice", charges.Held{SpreadClicks: 1})
	wallet := charges.NewCharges(store, spreadRules(1, false))
	spread, err := wallet.TakeSpread(t.Context(), "alice")
	require.NoError(t, err)

	require.NoError(t, wallet.GiveBackSpread(t.Context(), "alice", spread))
	require.NoError(t, wallet.GiveBackSpread(t.Context(), "alice", charges.Spread{Rings: 1}), "nothing was spent")

	assert.Equal(t, 1, store.Kept("alice").SpreadClicks)
}
