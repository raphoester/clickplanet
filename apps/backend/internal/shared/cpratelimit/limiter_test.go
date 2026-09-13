package cpratelimit

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func allow(l *Limiter, key string) bool {
	allowed, _ := l.Take(key)
	return allowed
}

func newTestLimiter() (*Limiter, *cptime.FixedClock) {
	clock := cptime.NewFixedClock(epoch)
	return New(Config{PerSecond: 1, Burst: 10}, clock), clock
}

func TestTakeSpendsTheBurstThenRefuses(t *testing.T) {
	limiter, _ := newTestLimiter()

	for i := 0; i < 10; i++ {
		require.Truef(t, allow(limiter, "1.2.3.4"), "click %d should be allowed", i)
	}

	require.False(t, allow(limiter, "1.2.3.4"))
}

func TestTakeNSpendsAllOrNothing(t *testing.T) {
	limiter, _ := newTestLimiter()

	for range 3 {
		allowed, _ := limiter.TakeN("1.2.3.4", 3)
		require.True(t, allowed)
	}

	allowed, state := limiter.TakeN("1.2.3.4", 3)
	require.False(t, allowed, "one token left does not pay for three")
	require.InDelta(t, 1.0, state.Tokens, 1e-9, "a refusal spends nothing")

	allowed, _ = limiter.TakeN("1.2.3.4", 1)
	require.True(t, allowed)
}

func TestRefillsAtTheConfiguredRate(t *testing.T) {
	limiter, clock := newTestLimiter()

	for i := 0; i < 10; i++ {
		require.True(t, allow(limiter, "1.2.3.4"))
	}

	clock.Advance(500 * time.Millisecond)
	require.False(t, allow(limiter, "1.2.3.4"), "half a token is not a click")

	clock.Advance(500 * time.Millisecond)
	require.True(t, allow(limiter, "1.2.3.4"))
	require.False(t, allow(limiter, "1.2.3.4"), "the token was just spent")
}

func TestRefillStopsAtTheBurst(t *testing.T) {
	limiter, clock := newTestLimiter()

	require.True(t, allow(limiter, "1.2.3.4"))
	clock.Advance(time.Hour)

	for i := 0; i < 10; i++ {
		require.Truef(t, allow(limiter, "1.2.3.4"), "click %d should be allowed", i)
	}

	require.False(t, allow(limiter, "1.2.3.4"), "an idle hour must not bank more than the burst")
}

func TestKeysAreIndependent(t *testing.T) {
	limiter, _ := newTestLimiter()

	for i := 0; i < 10; i++ {
		require.True(t, allow(limiter, "1.2.3.4"))
	}

	require.False(t, allow(limiter, "1.2.3.4"))
	require.True(t, allow(limiter, "5.6.7.8"))
}

func TestAClockGoingBackwardsTakesNoTokens(t *testing.T) {
	limiter, clock := newTestLimiter()

	require.True(t, allow(limiter, "1.2.3.4"))
	clock.Advance(-time.Hour)

	for i := 0; i < 9; i++ {
		require.Truef(t, allow(limiter, "1.2.3.4"), "click %d should be allowed", i)
	}

	require.False(t, allow(limiter, "1.2.3.4"))
}

func TestSweepForgetsOnlyTheRefilledBuckets(t *testing.T) {
	limiter, clock := newTestLimiter()

	require.True(t, allow(limiter, "idle"))
	for i := 0; i < 10; i++ {
		require.True(t, allow(limiter, "busy"))
	}

	limiter.sweep()
	require.Len(t, limiter.buckets, 2, "neither bucket has refilled yet")

	clock.Advance(time.Second)
	limiter.sweep()

	require.NotContains(t, limiter.buckets, "idle")
	require.Contains(t, limiter.buckets, "busy")

	for i := 0; i < 10; i++ {
		require.Truef(t, allow(limiter, "idle"), "click %d should be allowed", i)
	}
	require.False(t, allow(limiter, "idle"))

	require.True(t, allow(limiter, "busy"))
	require.False(t, allow(limiter, "busy"))
}

func TestDefaultsApplyToAZeroConfig(t *testing.T) {
	limiter := New(Config{}, cptime.NewFixedClock(epoch))

	require.InDelta(t, float64(defaultPerSecond), limiter.config.PerSecond, 1e-9)
	require.Equal(t, defaultBurst, limiter.config.Burst)
	require.Equal(t, defaultSweepInterval, limiter.config.SweepInterval)
}

func TestTakeIsSafeUnderConcurrentCallers(t *testing.T) {
	limiter, _ := newTestLimiter()

	const callers = 50
	allowed := make(chan bool, callers)

	wg := sync.WaitGroup{}
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed <- allow(limiter, "1.2.3.4")
		}()
	}
	wg.Wait()
	close(allowed)

	granted := 0
	for ok := range allowed {
		if ok {
			granted++
		}
	}

	require.Equal(t, 10, granted, "the burst is the whole allowance, however many goroutines ask")
}

func TestTakeReportsWhatIsLeftAndThePolicyToReplayIt(t *testing.T) {
	limiter, clock := newTestLimiter()

	_, state := limiter.Take("1.2.3.4")
	require.InDelta(t, float64(9), state.Tokens, 1e-9)
	require.Equal(t, 10, state.Capacity)
	require.InDelta(t, float64(1), state.PerSecond, 1e-9)

	clock.Advance(500 * time.Millisecond)

	_, state = limiter.Take("1.2.3.4")
	require.InDelta(t, 8.5, state.Tokens, 1e-9, "the half second refilled before the token was spent")
}

func TestARefusedCallerStillLearnsHowLongTheWaitIs(t *testing.T) {
	limiter, clock := newTestLimiter()

	for i := 0; i < 10; i++ {
		require.True(t, allow(limiter, "1.2.3.4"))
	}

	clock.Advance(400 * time.Millisecond)

	allowed, state := limiter.Take("1.2.3.4")
	require.False(t, allowed)
	require.InDelta(t, 0.4, state.Tokens, 1e-9, "the wait is readable off the fraction")
}

func TestPeekSpendsNothing(t *testing.T) {
	limiter, _ := newTestLimiter()

	require.True(t, allow(limiter, "1.2.3.4"))

	for i := 0; i < 5; i++ {
		require.InDelta(t, float64(9), limiter.Peek("1.2.3.4").Tokens, 1e-9)
	}

	for i := 0; i < 9; i++ {
		require.Truef(t, allow(limiter, "1.2.3.4"), "click %d should be allowed", i)
	}
	require.False(t, allow(limiter, "1.2.3.4"))
}

func TestPeekingAtAnUnknownKeyRemembersNothing(t *testing.T) {
	limiter, _ := newTestLimiter()

	require.InDelta(t, float64(10), limiter.Peek("1.2.3.4").Tokens, 1e-9)

	require.Empty(t, limiter.buckets, "reading an allowance must not create one")
}

func TestPeekRefillsBeforeReporting(t *testing.T) {
	limiter, clock := newTestLimiter()

	for i := 0; i < 10; i++ {
		require.True(t, allow(limiter, "1.2.3.4"))
	}

	clock.Advance(3 * time.Second)

	require.InDelta(t, float64(3), limiter.Peek("1.2.3.4").Tokens, 1e-9)
}
