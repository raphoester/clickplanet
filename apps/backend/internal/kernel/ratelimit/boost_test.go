package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const key = "1.2.3.4"

func TestABoostWidensTheAllowanceAndTheRate(t *testing.T) {
	limiter, clock := newTestLimiter()

	state := limiter.Boost(key, 3, clock.Now().Add(time.Minute))

	assert.Equal(t, 30, state.Capacity)
	assert.InDelta(t, 3.0, state.PerSecond, 1e-9)
}

func TestABoostLeavesTheTokensAlreadyInHandWhereTheyAre(t *testing.T) {
	limiter, clock := newTestLimiter()

	for i := 0; i < 6; i++ {
		require.True(t, allow(limiter, key))
	}

	// Four left of ten. A boost widens the ceiling and the rate; it does not
	// hand out a full bucket.
	state := limiter.Boost(key, 3, clock.Now().Add(time.Minute))

	assert.InDelta(t, 4.0, state.Tokens, 1e-9)
	assert.Equal(t, 30, state.Capacity)
}

func TestABoostedBucketRefillsAtTheMultipliedRate(t *testing.T) {
	limiter, clock := newTestLimiter()

	for i := 0; i < 10; i++ {
		require.True(t, allow(limiter, key))
	}

	limiter.Boost(key, 3, clock.Now().Add(time.Minute))
	clock.advance(2 * time.Second)

	// Two seconds at 3/s rather than at 1/s.
	assert.InDelta(t, 6.0, limiter.Peek(key).Tokens, 1e-9)
}

func TestTheRefillIsSplitAtTheMomentTheBoostLapses(t *testing.T) {
	limiter, clock := newTestLimiter()

	for i := 0; i < 10; i++ {
		require.True(t, allow(limiter, key))
	}

	limiter.Boost(key, 3, clock.Now().Add(2*time.Second))

	// One reading that straddles the end: two seconds boosted, then two plain.
	// Paid entirely at either rate this would be 12 or 4, and both are wrong.
	clock.advance(4 * time.Second)

	assert.InDelta(t, 8.0, limiter.Peek(key).Tokens, 1e-9)
}

func TestTheAllowanceComesBackDownWhenTheBoostEnds(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.Boost(key, 3, clock.Now().Add(time.Second))
	clock.advance(10 * time.Second)

	state := limiter.Peek(key)

	// Both the ceiling and the rate are the plain ones again, and the tokens
	// banked above the burst are gone with it — a bucket still holding thirty
	// would spend the boost long after it was over.
	assert.Equal(t, 10, state.Capacity)
	assert.InDelta(t, 1.0, state.PerSecond, 1e-9)
	assert.LessOrEqual(t, state.Tokens, 10.0)
}

func TestABoostedCallerMaySpendPastThePlainBurst(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.Boost(key, 3, clock.Now().Add(time.Minute))
	clock.advance(30 * time.Second)

	spent := 0
	for {
		if !allow(limiter, key) {
			break
		}
		spent++
	}

	assert.Equal(t, 30, spent)
}

func TestTheSweepDoesNotForgetABoostStillRunning(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.Boost(key, 3, clock.Now().Add(time.Hour))
	clock.advance(time.Minute)

	// A boosted bucket sits above the plain burst, which is exactly what the
	// sweep treats as "same as a fresh one". Forgetting it would end the boost.
	limiter.sweep()

	assert.Equal(t, 30, limiter.Peek(key).Capacity)
}

func TestTheSweepStillForgetsABucketWhoseBoostIsOver(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.Boost(key, 3, clock.Now().Add(time.Second))
	clock.advance(time.Hour)

	limiter.sweep()

	assert.Empty(t, limiter.buckets)
}

func TestBoostingOneCallerLeavesEveryOtherAlone(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.Boost(key, 3, clock.Now().Add(time.Minute))

	assert.Equal(t, 10, limiter.Peek("5.6.7.8").Capacity)
}

func TestAnExpiredOrAbsentBoostChangesNothing(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.Boost(key, 3, clock.Now().Add(-time.Second))
	limiter.Boost("5.6.7.8", 1, clock.Now().Add(time.Hour))

	assert.Equal(t, 10, limiter.Peek(key).Capacity)
	assert.Equal(t, 10, limiter.Peek("5.6.7.8").Capacity)
}

func TestASecondBoostReplacesTheFirst(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.Boost(key, 3, clock.Now().Add(time.Second))
	state := limiter.Boost(key, 2, clock.Now().Add(time.Hour))

	assert.Equal(t, 20, state.Capacity)
}

func TestTheBoostIsOverAtItsDeadlineRatherThanAfterIt(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.Boost(key, 3, clock.Now().Add(time.Minute))

	clock.advance(time.Minute - time.Millisecond)
	require.Equal(t, 30, limiter.Peek(key).Capacity, "still running a millisecond short of the deadline")

	clock.advance(time.Millisecond)
	require.Equal(t, 10, limiter.Peek(key).Capacity, "over on the deadline itself")
}
