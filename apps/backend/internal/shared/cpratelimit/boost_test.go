package cpratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const boostKey = "1.2.3.4"

func TestABoostSpeedsUpTheRateAndLeavesTheBankItsSize(t *testing.T) {
	limiter, clock := newTestLimiter()

	state := limiter.Boost(boostKey, 3, clock.Now().Add(time.Minute))

	assert.Equal(t, 10, state.Capacity)
	assert.InDelta(t, 3.0, state.PerSecond, 1e-9)
}

func TestABoostLeavesTheTokensAlreadyInHandWhereTheyAre(t *testing.T) {
	limiter, clock := newTestLimiter()

	for i := 0; i < 6; i++ {
		require.True(t, allow(limiter, boostKey))
	}

	// Four left of ten. A boost speeds up the rate; it does not hand out a full
	// bucket.
	state := limiter.Boost(boostKey, 3, clock.Now().Add(time.Minute))

	assert.InDelta(t, 4.0, state.Tokens, 1e-9)
}

func TestABoostedBucketRefillsAtTheMultipliedRate(t *testing.T) {
	limiter, clock := newTestLimiter()

	for i := 0; i < 10; i++ {
		require.True(t, allow(limiter, boostKey))
	}

	limiter.Boost(boostKey, 3, clock.Now().Add(time.Minute))
	clock.Advance(2 * time.Second)

	// Two seconds at 3/s rather than at 1/s.
	assert.InDelta(t, 6.0, limiter.Peek(Key{Name: boostKey}).Tokens, 1e-9)
}

func TestTheRefillIsSplitAtTheMomentTheBoostLapses(t *testing.T) {
	limiter, clock := newTestLimiter()

	for i := 0; i < 10; i++ {
		require.True(t, allow(limiter, boostKey))
	}

	limiter.Boost(boostKey, 3, clock.Now().Add(2*time.Second))

	// One reading that straddles the end: two seconds boosted, then two plain.
	// Paid entirely at either rate this would be 12 or 4, and both are wrong.
	clock.Advance(4 * time.Second)

	assert.InDelta(t, 8.0, limiter.Peek(Key{Name: boostKey}).Tokens, 1e-9)
}

func TestTheRateComesBackDownWhenTheBoostEnds(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.Boost(boostKey, 3, clock.Now().Add(time.Second))
	clock.Advance(10 * time.Second)

	state := limiter.Peek(Key{Name: boostKey})

	assert.Equal(t, 10, state.Capacity)
	assert.InDelta(t, 1.0, state.PerSecond, 1e-9)
	assert.False(t, state.Boosted)
}

func TestABoostedCallerNeverBanksPastTheBurst(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.Boost(boostKey, 3, clock.Now().Add(time.Minute))
	clock.Advance(30 * time.Second)

	spent := 0
	for allow(limiter, boostKey) {
		spent++
	}

	assert.Equal(t, 10, spent)
}

func TestTheSweepDoesNotForgetABoostStillRunning(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.Boost(boostKey, 3, clock.Now().Add(time.Hour))
	clock.Advance(time.Minute)

	// A full boosted bucket is what the sweep treats as "same as a fresh one".
	// Forgetting it would end the boost.
	limiter.sweep()

	assert.InDelta(t, 3.0, limiter.Peek(Key{Name: boostKey}).PerSecond, 1e-9)
}

func TestTheSweepStillForgetsABucketWhoseBoostIsOver(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.Boost(boostKey, 3, clock.Now().Add(time.Second))
	clock.Advance(time.Hour)

	limiter.sweep()

	assert.Empty(t, limiter.buckets)
}

func TestBoostingOneCallerLeavesEveryOtherAlone(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.Boost(boostKey, 3, clock.Now().Add(time.Minute))

	assert.InDelta(t, 1.0, limiter.Peek(Key{Name: "5.6.7.8"}).PerSecond, 1e-9)
}

func TestAnExpiredOrAbsentBoostChangesNothing(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.Boost(boostKey, 3, clock.Now().Add(-time.Second))
	limiter.Boost("5.6.7.8", 1, clock.Now().Add(time.Hour))

	assert.InDelta(t, 1.0, limiter.Peek(Key{Name: boostKey}).PerSecond, 1e-9)
	assert.InDelta(t, 1.0, limiter.Peek(Key{Name: "5.6.7.8"}).PerSecond, 1e-9)
}

func TestASecondBoostReplacesTheFirst(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.Boost(boostKey, 3, clock.Now().Add(time.Second))
	state := limiter.Boost(boostKey, 2, clock.Now().Add(time.Hour))

	assert.InDelta(t, 2.0, state.PerSecond, 1e-9)
}

func TestTheBoostIsOverAtItsDeadlineRatherThanAfterIt(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.Boost(boostKey, 3, clock.Now().Add(time.Minute))

	clock.Advance(time.Minute - time.Millisecond)
	require.InDelta(t, 3.0, limiter.Peek(Key{Name: boostKey}).PerSecond, 1e-9, "still running a millisecond short of the deadline")

	clock.Advance(time.Millisecond)
	require.InDelta(t, 1.0, limiter.Peek(Key{Name: boostKey}).PerSecond, 1e-9, "over on the deadline itself")
}

func TestTheReadingSaysWhetherABoostRuns(t *testing.T) {
	limiter, clock := newTestLimiter()

	_, plain := limiter.Take(boostKey)
	assert.False(t, plain.Boosted)

	assert.True(t, limiter.Boost(boostKey, 3, clock.Now().Add(time.Minute)).Boosted)
	_, boosted := limiter.Take(boostKey)
	assert.True(t, boosted.Boosted)
	assert.False(t, limiter.Peek(Key{Name: "5.6.7.8"}).Boosted)

	clock.Advance(time.Minute)
	_, over := limiter.Take(boostKey)
	assert.False(t, over.Boosted, "over on the deadline itself")
}

func TestABoostOnOneBucketDoesNotSpeedUpAnotherSpentWithIt(t *testing.T) {
	limiter, _ := newTestLimiter()
	account, scope := Key{Name: "account"}, Key{Name: "scope", Scale: 10}

	limiter.Boost(account.Name, 3, epoch.Add(time.Minute))

	_, states := limiter.TakeAll(1, account, scope)
	assert.True(t, states[0].Boosted)
	assert.InDelta(t, 3.0, states[0].PerSecond, 1e-9)
	assert.False(t, states[1].Boosted)
	assert.InDelta(t, 10.0, states[1].PerSecond, 1e-9)
}

func TestAPaceMultipliesTheRateFromTheTakeOnAndLeavesTheBurst(t *testing.T) {
	limiter, clock := newTestLimiter()

	for i := 0; i < 5; i++ {
		require.True(t, allow(limiter, boostKey))
	}
	clock.Advance(2 * time.Second)

	// The two seconds before this take refill at the plain rate: the pace only
	// applies from the take that sets it.
	_, states := limiter.TakeAll(1, Key{Name: boostKey, Pace: 0.5})
	assert.InDelta(t, 6.0, states[0].Tokens, 1e-9)
	assert.InDelta(t, 0.5, states[0].PerSecond, 1e-9)
	assert.Equal(t, 10, states[0].Capacity)

	clock.Advance(2 * time.Second)
	assert.InDelta(t, 7.0, limiter.Peek(Key{Name: boostKey, Pace: 4}).Tokens, 1e-9, "a peek sets no pace")

	_, states = limiter.TakeAll(1, Key{Name: boostKey})
	assert.InDelta(t, 0.5, states[0].PerSecond, 1e-9, "a take with no pace keeps the one in force")
}

func TestAPaceAndABoostMultiply(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.TakeAll(1, Key{Name: boostKey, Pace: 2.0 / 3})
	state := limiter.Boost(boostKey, 3, clock.Now().Add(time.Minute))

	assert.InDelta(t, 2.0, state.PerSecond, 1e-9)
}
