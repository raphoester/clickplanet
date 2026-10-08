package tempo_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func TestPlainRulesChangeNothing(t *testing.T) {
	rules := tempo.Plain()

	assert.InDelta(t, 1.0, rules.RefillMultiplier(), 1e-9)
	_, scheduled := rules.BoxInterval()
	assert.False(t, scheduled)
	_, gifting := rules.Gift()
	assert.False(t, gifting)
	assert.False(t, rules.Frozen())
	require.NoError(t, rules.FrozenError())
}

func TestRulesHoldWhatTheyWereGiven(t *testing.T) {
	gift, err := tempo.GiftOf("finale-0", finale)
	require.NoError(t, err)

	rules, err := tempo.NewRules(3, 2*time.Minute, false)
	require.NoError(t, err)
	rules = rules.WithGift(gift)

	assert.InDelta(t, 3.0, rules.RefillMultiplier(), 1e-9)
	interval, scheduled := rules.BoxInterval()
	assert.True(t, scheduled)
	assert.Equal(t, 2*time.Minute, interval)
	held, gifting := rules.Gift()
	assert.True(t, gifting)
	assert.Equal(t, tempo.GiftTag("finale-0"), held.Tag())
	assert.Equal(t, finale, held.MadeBefore())
}

func TestAZeroMultiplierIsThePlainPace(t *testing.T) {
	rules, err := tempo.NewRules(0, 0, false)

	require.NoError(t, err)
	assert.InDelta(t, 1.0, rules.RefillMultiplier(), 1e-9)
}

func TestRulesNobodyCanPlayByAreRefused(t *testing.T) {
	for _, multiplier := range []float64{0.5, -1, math.NaN(), math.Inf(1)} {
		_, err := tempo.NewRules(multiplier, 0, false)
		require.ErrorIs(t, err, tempo.ErrInvalidRules, "multiplier %v", multiplier)
	}

	_, err := tempo.NewRules(1, -time.Second, false)
	require.ErrorIs(t, err, tempo.ErrInvalidRules)

	_, err = tempo.GiftOf("", finale)
	require.ErrorIs(t, err, tempo.ErrInvalidRules)
}

func TestAFrozenMapRefusesWrites(t *testing.T) {
	rules, err := tempo.NewRules(1, 0, true)

	require.NoError(t, err)
	assert.True(t, rules.Frozen())
	require.ErrorIs(t, rules.FrozenError(), tempo.ErrFrozen)
}

func TestAGiftIsOwedToAnAccountMadeBeforeItsCutoff(t *testing.T) {
	gift, err := tempo.GiftOf("finale-0", finale)
	require.NoError(t, err)

	assert.True(t, gift.OwedTo(clicks.Payer{Account: "old", Created: finale.Add(-time.Hour)}))
	assert.False(t, gift.OwedTo(clicks.Payer{Account: "fresh", Created: finale.Add(time.Second)}))
	assert.False(t, gift.OwedTo(clicks.Payer{Account: "fresh", Created: finale}), "made at the cutoff is not before it")
	assert.True(t, gift.OwedTo(clicks.Payer{Account: "older-than-dated-ids"}), "an id that says nothing is older than those that do")
	assert.False(t, gift.OwedTo(clicks.Payer{Scope: "1.2.3.4"}), "nobody without an account")

	anyone, err := tempo.GiftOf("finale-0", time.Time{})
	require.NoError(t, err)
	assert.True(t, anyone.OwedTo(clicks.Payer{Account: "fresh", Created: finale.Add(time.Hour)}), "no cutoff is every account")
}
