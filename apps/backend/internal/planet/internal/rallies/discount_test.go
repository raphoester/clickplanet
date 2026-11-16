package rallies_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func discountOver(t *testing.T, store *rallies.MemoryStore, clock *cptime.FixedClock) rallies.Discount {
	t.Helper()
	return rallies.NewDiscount(store, rallies.Line{}, charges.TestRules, clock)
}

func keep(t *testing.T, store *rallies.MemoryStore, rally rallies.Rally) {
	t.Helper()
	account := rally.TakeSnapshot().Account
	require.NoError(t, store.Change(t.Context(), account, func(rallies.Rally) (rallies.Rally, error) { return rally, nil }))
}

func costOf(t *testing.T, discount rallies.Discount, flag string, tile uint32) float64 {
	t.Helper()
	cost, err := discount.Cost(t.Context(), flag, tile)
	require.NoError(t, err)
	return cost
}

func TestAClickInsideARingOfItsFlagCostsTheRallysPrice(t *testing.T) {
	store, clock := rallies.NewMemoryStore(), cptime.NewFixedClock(calledAt.Add(time.Minute))
	keep(t, store, rallyOf(alice, "fr", 10, calledAt))
	discount := discountOver(t, store, clock)

	assert.InDelta(t, 0.5, costOf(t, discount, "fr", 12), 1e-12)
	assert.InDelta(t, 1.0, costOf(t, discount, "fr", 30), 1e-12, "outside the ring")
	assert.InDelta(t, 1.0, costOf(t, discount, "de", 12), 1e-12, "another flag's click gets nothing of it")
}

func TestRingsThatOverlapCostNoLessThanOne(t *testing.T) {
	store, clock := rallies.NewMemoryStore(), cptime.NewFixedClock(calledAt.Add(time.Minute))
	keep(t, store, rallyOf(alice, "fr", 10, calledAt))
	keep(t, store, rallyOf(bob, "fr", 12, calledAt))

	assert.InDelta(t, 0.5, costOf(t, discountOver(t, store, clock), "fr", 11), 1e-12, "never a quarter")
}

func TestARingThatEndedCostsAWholeClick(t *testing.T) {
	store, clock := rallies.NewMemoryStore(), cptime.NewFixedClock(calledAt.Add(time.Minute))
	keep(t, store, rallyOf(alice, "fr", 10, calledAt))
	discount := discountOver(t, store, clock)

	clock.Advance(2 * time.Minute)

	assert.InDelta(t, 1.0, costOf(t, discount, "fr", 10), 1e-12)
}

func TestTheCostIsReadFromTheRulesAtEachClick(t *testing.T) {
	store, clock := rallies.NewMemoryStore(), cptime.NewFixedClock(calledAt)
	keep(t, store, rallyOf(alice, "fr", 10, calledAt))
	rules := charges.TestRules()
	discount := rallies.NewDiscount(store, rallies.Line{}, func() charges.Rules { return rules }, clock)

	rules.Rally.Cost = 0.25

	assert.InDelta(t, 0.25, costOf(t, discount, "fr", 10), 1e-12)
}

func TestAFailedReadCostsAWholeClickAndSaysWhy(t *testing.T) {
	store := rallies.NewMemoryStore()
	store.FailWith(errors.New("postgres down"))

	cost, err := discountOver(t, store, cptime.NewFixedClock(calledAt)).Cost(t.Context(), "fr", 10)

	require.ErrorIs(t, err, rallies.ErrUnread)
	assert.InDelta(t, 1.0, cost, 1e-12)
}
