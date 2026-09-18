package cpratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const key = "1.2.3.4"

func TestAPaceMultipliesTheRateFromTheTakeOnAndLeavesTheBurst(t *testing.T) {
	limiter, clock := newTestLimiter()

	for i := 0; i < 5; i++ {
		require.True(t, allow(limiter, key))
	}
	clock.Advance(2 * time.Second)

	// The two seconds before this take refill at the plain rate: the pace only
	// applies from the take that sets it.
	_, states := limiter.TakeAll(1, Key{Name: key, Pace: 0.5})
	assert.InDelta(t, 6.0, states[0].Tokens, 1e-9)
	assert.InDelta(t, 0.5, states[0].PerSecond, 1e-9)
	assert.Equal(t, 10, states[0].Capacity)

	clock.Advance(2 * time.Second)
	assert.InDelta(t, 7.0, limiter.Peek(Key{Name: key, Pace: 4}).Tokens, 1e-9, "a peek sets no pace")

	_, states = limiter.TakeAll(1, Key{Name: key})
	assert.InDelta(t, 0.5, states[0].PerSecond, 1e-9, "a take with no pace keeps the one in force")
}

func TestFillTopsTheBucketUpToItsCapacity(t *testing.T) {
	limiter, _ := newTestLimiter()
	for range 7 {
		require.True(t, allow(limiter, key))
	}

	filled, state := limiter.Fill(Key{Name: key})

	require.True(t, filled)
	assert.InDelta(t, 10.0, state.Tokens, 1e-9)
	assert.Equal(t, 10, state.Capacity, "a fill never banks past the burst")
}

func TestFillLeavesAFullBucketAloneAndSaysSo(t *testing.T) {
	limiter, clock := newTestLimiter()
	require.True(t, allow(limiter, key))
	clock.Advance(time.Minute)

	filled, state := limiter.Fill(Key{Name: key})

	assert.False(t, filled, "it refilled on its own, so there was nothing to fill")
	assert.InDelta(t, 10.0, state.Tokens, 1e-9)
}

func TestFillDoesNotMakeTheLimiterRememberAnUnknownKey(t *testing.T) {
	limiter, _ := newTestLimiter()

	filled, state := limiter.Fill(Key{Name: key})

	assert.False(t, filled, "an unknown key is a full bucket")
	assert.InDelta(t, 10.0, state.Tokens, 1e-9)
	assert.Empty(t, limiter.buckets)
}

func TestFillingOneKeyLeavesEveryOtherAlone(t *testing.T) {
	limiter, _ := newTestLimiter()
	for range 5 {
		require.True(t, allow(limiter, key))
		require.True(t, allow(limiter, "other"))
	}

	_, _ = limiter.Fill(Key{Name: key})

	assert.InDelta(t, 5.0, limiter.Peek(Key{Name: "other"}).Tokens, 1e-9)
}

func TestAFilledBucketKeepsItsPace(t *testing.T) {
	limiter, _ := newTestLimiter()
	_, _ = limiter.TakeAll(1, Key{Name: key, Pace: 0.5})

	_, state := limiter.Fill(Key{Name: key})

	assert.InDelta(t, 0.5, state.PerSecond, 1e-9)
}
