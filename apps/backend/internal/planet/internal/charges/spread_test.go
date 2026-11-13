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

	rings, err := wallet.TakeSpread(t.Context(), "alice")

	require.NoError(t, err)
	assert.Equal(t, 2, rings)
	assert.Equal(t, 1, store.Kept("alice").SpreadClicks)
}

func TestWithNoSpreadClickHeldAClickSpreadsNothing(t *testing.T) {
	wallet := charges.NewCharges(charges.NewMemoryStore(), spreadRules(2, false))

	rings, err := wallet.TakeSpread(t.Context(), "alice")
	require.NoError(t, err)
	assert.Zero(t, rings)

	guest, err := wallet.TakeSpread(t.Context(), charges.NoHolder)
	require.NoError(t, err)
	assert.Zero(t, guest)
}

func TestAnUnlimitedSpreadSpendsNothingAndNeedsNothing(t *testing.T) {
	store := charges.NewMemoryStore()
	store.Keep("alice", charges.Held{SpreadClicks: 3})
	wallet := charges.NewCharges(store, spreadRules(2, true))

	held, err := wallet.TakeSpread(t.Context(), "alice")
	require.NoError(t, err)
	guest, err := wallet.TakeSpread(t.Context(), charges.NoHolder)
	require.NoError(t, err)

	assert.Equal(t, 2, held)
	assert.Equal(t, 2, guest)
	assert.Equal(t, 3, store.Kept("alice").SpreadClicks, "the pool held is left for when the set stops")
}
