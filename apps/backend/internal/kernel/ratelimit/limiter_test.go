package ratelimit

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestLimiter() (*Limiter, *fakeClock) {
	clock := &fakeClock{now: epoch}
	return New(Config{PerSecond: 1, Burst: 10}, clock), clock
}

func TestAllowSpendsTheBurstThenRefuses(t *testing.T) {
	limiter, _ := newTestLimiter()

	for i := 0; i < 10; i++ {
		require.Truef(t, limiter.Allow("1.2.3.4"), "click %d should be allowed", i)
	}

	require.False(t, limiter.Allow("1.2.3.4"))
}

func TestRefillsAtTheConfiguredRate(t *testing.T) {
	limiter, clock := newTestLimiter()

	for i := 0; i < 10; i++ {
		require.True(t, limiter.Allow("1.2.3.4"))
	}

	clock.advance(500 * time.Millisecond)
	require.False(t, limiter.Allow("1.2.3.4"), "half a token is not a click")

	clock.advance(500 * time.Millisecond)
	require.True(t, limiter.Allow("1.2.3.4"))
	require.False(t, limiter.Allow("1.2.3.4"), "the token was just spent")
}

func TestRefillStopsAtTheBurst(t *testing.T) {
	limiter, clock := newTestLimiter()

	require.True(t, limiter.Allow("1.2.3.4"))
	clock.advance(time.Hour)

	for i := 0; i < 10; i++ {
		require.Truef(t, limiter.Allow("1.2.3.4"), "click %d should be allowed", i)
	}

	require.False(t, limiter.Allow("1.2.3.4"), "an idle hour must not bank more than the burst")
}

func TestKeysAreIndependent(t *testing.T) {
	limiter, _ := newTestLimiter()

	for i := 0; i < 10; i++ {
		require.True(t, limiter.Allow("1.2.3.4"))
	}

	require.False(t, limiter.Allow("1.2.3.4"))
	require.True(t, limiter.Allow("5.6.7.8"))
}

func TestAClockGoingBackwardsTakesNoTokens(t *testing.T) {
	limiter, clock := newTestLimiter()

	require.True(t, limiter.Allow("1.2.3.4"))
	clock.advance(-time.Hour)

	for i := 0; i < 9; i++ {
		require.Truef(t, limiter.Allow("1.2.3.4"), "click %d should be allowed", i)
	}

	require.False(t, limiter.Allow("1.2.3.4"))
}

func TestSweepForgetsOnlyTheRefilledBuckets(t *testing.T) {
	limiter, clock := newTestLimiter()

	require.True(t, limiter.Allow("idle"))
	for i := 0; i < 10; i++ {
		require.True(t, limiter.Allow("busy"))
	}

	limiter.sweep()
	require.Len(t, limiter.buckets, 2, "neither bucket has refilled yet")

	clock.advance(time.Second)
	limiter.sweep()

	require.NotContains(t, limiter.buckets, "idle")
	require.Contains(t, limiter.buckets, "busy")

	for i := 0; i < 10; i++ {
		require.Truef(t, limiter.Allow("idle"), "click %d should be allowed", i)
	}
	require.False(t, limiter.Allow("idle"))

	require.True(t, limiter.Allow("busy"))
	require.False(t, limiter.Allow("busy"))
}

func TestDefaultsApplyToAZeroConfig(t *testing.T) {
	limiter := New(Config{}, &fakeClock{now: epoch})

	require.Equal(t, float64(defaultPerSecond), limiter.config.PerSecond)
	require.Equal(t, defaultBurst, limiter.config.Burst)
	require.Equal(t, defaultSweepInterval, limiter.config.SweepInterval)
}

func TestAllowIsSafeUnderConcurrentCallers(t *testing.T) {
	limiter, _ := newTestLimiter()

	const callers = 50
	allowed := make(chan bool, callers)

	wg := sync.WaitGroup{}
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed <- limiter.Allow("1.2.3.4")
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
