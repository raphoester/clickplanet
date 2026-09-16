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
	return New("test", Config{PerSecond: 1, Burst: 10}, clock), clock
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

	for range 6 {
		allowed, _ := limiter.TakeN("1.2.3.4", 1.5)
		require.True(t, allowed)
	}

	allowed, state := limiter.TakeN("1.2.3.4", 1.5)
	require.False(t, allowed, "one token left does not pay for one and a half")
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
	limiter := New("test", Config{}, cptime.NewFixedClock(epoch))

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
		require.InDelta(t, float64(9), limiter.Peek(Key{Name: "1.2.3.4"}).Tokens, 1e-9)
	}

	for i := 0; i < 9; i++ {
		require.Truef(t, allow(limiter, "1.2.3.4"), "click %d should be allowed", i)
	}
	require.False(t, allow(limiter, "1.2.3.4"))
}

func TestPeekingAtAnUnknownKeyRemembersNothing(t *testing.T) {
	limiter, _ := newTestLimiter()

	require.InDelta(t, float64(10), limiter.Peek(Key{Name: "1.2.3.4"}).Tokens, 1e-9)

	require.Empty(t, limiter.buckets, "reading an allowance must not create one")
}

func TestPeekRefillsBeforeReporting(t *testing.T) {
	limiter, clock := newTestLimiter()

	for i := 0; i < 10; i++ {
		require.True(t, allow(limiter, "1.2.3.4"))
	}

	clock.Advance(3 * time.Second)

	require.InDelta(t, float64(3), limiter.Peek(Key{Name: "1.2.3.4"}).Tokens, 1e-9)
}

func TestAScaledBucketHoldsAndRefillsItsScale(t *testing.T) {
	limiter, clock := newTestLimiter()
	campus := Key{Name: "campus", Scale: 10}

	allowed, states := limiter.TakeAll(100, campus)
	require.True(t, allowed, "a scale of ten starts with ten bursts")
	require.Equal(t, 100, states[0].Capacity)
	require.InDelta(t, 10.0, states[0].PerSecond, 1e-9)

	clock.Advance(time.Second)
	require.InDelta(t, 10.0, limiter.Peek(campus).Tokens, 1e-9, "and refills ten a second")
}

func TestTakeAllSpendsFromEveryBucketOrFromNone(t *testing.T) {
	limiter, _ := newTestLimiter()
	account, scope := Key{Name: "account"}, Key{Name: "scope", Scale: 10}

	for range 10 {
		allowed, _ := limiter.TakeAll(1, account, scope)
		require.True(t, allowed)
	}

	allowed, states := limiter.TakeAll(1, account, scope)
	require.False(t, allowed, "the account bucket is empty")
	require.InDelta(t, 0.0, states[0].Tokens, 1e-9)
	require.InDelta(t, 90.0, states[1].Tokens, 1e-9, "the refusal spent nothing from the scope bucket")

	allowed, _ = limiter.TakeAll(1, Key{Name: "another account"}, scope)
	require.True(t, allowed, "another account on the scope still has its own allowance")
}

func TestTheSweepForgetsAScaledBucketOnlyOnceItIsFull(t *testing.T) {
	limiter, clock := newTestLimiter()
	campus := Key{Name: "campus", Scale: 10}

	limiter.TakeAll(20, campus)
	clock.Advance(time.Second)
	limiter.sweep()
	require.Contains(t, limiter.buckets, "campus", "90 tokens under a burst of 100 is not full")

	clock.Advance(time.Second)
	limiter.sweep()
	require.NotContains(t, limiter.buckets, "campus")
}
