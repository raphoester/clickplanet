package cpratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDrainTakesFromAFullBucket(t *testing.T) {
	limiter, _ := newTestLimiter()

	drained := limiter.Drain(4, Key{Name: key})

	require.Len(t, drained, 1)
	assert.InDelta(t, 4.0, drained[0].Taken, 1e-9)
	assert.InDelta(t, 6.0, drained[0].State.Tokens, 1e-9)
	assert.Equal(t, 10, drained[0].State.Capacity)
}

func TestDrainGoesBelowEmpty(t *testing.T) {
	limiter, _ := newTestLimiter()
	for range 8 {
		require.True(t, allow(limiter, key))
	}

	drained := limiter.Drain(5, Key{Name: key})

	assert.InDelta(t, 5.0, drained[0].Taken, 1e-9)
	assert.InDelta(t, -3.0, drained[0].State.Tokens, 1e-9, "a debt, not an empty bank")
}

func TestDrainStopsAtMinusOneBank(t *testing.T) {
	limiter, _ := newTestLimiter()
	for range 10 {
		require.True(t, allow(limiter, key))
	}
	_ = limiter.Drain(8, Key{Name: key})

	drained := limiter.Drain(5, Key{Name: key})

	assert.InDelta(t, 2.0, drained[0].Taken, 1e-9, "only what is left above minus the capacity")
	assert.InDelta(t, -10.0, drained[0].State.Tokens, 1e-9)

	drained = limiter.Drain(5, Key{Name: key})
	assert.InDelta(t, 0.0, drained[0].Taken, 1e-9, "a bank at the floor loses nothing more")
	assert.InDelta(t, -10.0, drained[0].State.Tokens, 1e-9)
}

func TestADebtRefillsAndHoldsEveryTakeBackUntilItIsPaid(t *testing.T) {
	limiter, clock := newTestLimiter()
	for range 10 {
		require.True(t, allow(limiter, key))
	}
	_ = limiter.Drain(4, Key{Name: key})

	clock.Advance(4500 * time.Millisecond)
	assert.InDelta(t, 0.5, limiter.Peek(Key{Name: key}).Tokens, 1e-9, "the refill runs from below zero")
	assert.False(t, allow(limiter, key), "half a click is not a click")

	clock.Advance(500 * time.Millisecond)
	assert.True(t, allow(limiter, key))
}

func TestDrainMakesAndKeepsABucketItDoesNotKnow(t *testing.T) {
	limiter, clock := newTestLimiter()

	_ = limiter.Drain(15, Key{Name: key})
	limiter.sweep()

	assert.Contains(t, limiter.buckets, key, "the sweep keeps a bucket in debt")
	clock.Advance(time.Second)
	assert.InDelta(t, -4.0, limiter.Peek(Key{Name: key}).Tokens, 1e-9)
}

func TestDrainReadsANewAccountsBankFromItsStart(t *testing.T) {
	limiter, clock := newTestLimiter()
	clock.Advance(3 * time.Second)

	drained := limiter.Drain(4, Key{Name: key, Since: epoch, Start: 2})

	assert.InDelta(t, 4.0, drained[0].Taken, 1e-9)
	assert.InDelta(t, 1.0, drained[0].State.Tokens, 1e-9, "2 at the start, 3 earned since, 4 drained")
}

func TestDrainTakesFromEveryBucketAtItsOwnScale(t *testing.T) {
	limiter, _ := newTestLimiter()

	drained := limiter.Drain(25, Key{Name: "account:a", Scale: 1}, Key{Name: "guests:1.2.3.4", Scale: 3})

	assert.InDelta(t, 20.0, drained[0].Taken, 1e-9, "a bank of 10 goes down to -10")
	assert.InDelta(t, -10.0, drained[0].State.Tokens, 1e-9)
	assert.InDelta(t, 25.0, drained[1].Taken, 1e-9)
	assert.InDelta(t, 5.0, drained[1].State.Tokens, 1e-9)
	assert.Equal(t, 30, drained[1].State.Capacity)
}

func TestDrainSetsNoPace(t *testing.T) {
	limiter, _ := newTestLimiter()
	_, _ = limiter.TakeAll(1, Key{Name: key, Pace: 0.5})

	drained := limiter.Drain(1, Key{Name: key, Pace: 4})

	assert.InDelta(t, 0.5, drained[0].State.PerSecond, 1e-9)
}

func TestDrainNeverPaysADebtBackAfterTheBurstShrank(t *testing.T) {
	burst := 10
	limiter := NewRuled("test", func() Policy { return Policy{PerSecond: 1, Burst: burst, Tempo: 1} }, time.Minute, nil)
	for range 10 {
		require.True(t, allow(limiter, key))
	}
	_ = limiter.Drain(10, Key{Name: key})
	burst = 4

	drained := limiter.Drain(1, Key{Name: key})

	assert.InDelta(t, 0.0, drained[0].Taken, 1e-9)
	assert.Less(t, drained[0].State.Tokens, -9.0, "a drain that would raise the bank to the new floor leaves it")
}

func TestATakeRefusesABankInDebt(t *testing.T) {
	limiter, _ := newTestLimiter()
	_ = limiter.Drain(12, Key{Name: key})

	allowed, states := limiter.TakeAll(1, Key{Name: key})

	assert.False(t, allowed)
	assert.InDelta(t, -2.0, states[0].Tokens, 1e-9, "a refused take spends nothing")
}

func TestAFillPaysTheWholeDebt(t *testing.T) {
	limiter, _ := newTestLimiter()
	_ = limiter.Drain(15, Key{Name: key})

	filled, state := limiter.Fill(Key{Name: key})

	assert.True(t, filled)
	assert.InDelta(t, 10.0, state.Tokens, 1e-9)
}
