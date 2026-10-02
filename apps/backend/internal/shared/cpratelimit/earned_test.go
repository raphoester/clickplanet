package cpratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestANewcomerEarnsItsBankInsteadOfBeingHandedIt(t *testing.T) {
	limiter, clock := newTestLimiter()
	newcomer := Key{Name: "account", Since: clock.Now(), Start: 2}

	assert.InDelta(t, 2.0, limiter.Peek(newcomer).Tokens, 1e-9)
	assert.Empty(t, limiter.buckets, "reading it remembers nothing")

	clock.Advance(3 * time.Second)
	for i := range 5 {
		allowed, _ := limiter.TakeAll(1, newcomer)
		require.Truef(t, allowed, "click %d", i)
	}

	allowed, states := limiter.TakeAll(1, newcomer)
	assert.False(t, allowed, "two to start and three earned is five")
	assert.Equal(t, 10, states[0].Capacity, "the bank keeps its size")
}

func TestANewcomerThatArrivedLongAgoIsFull(t *testing.T) {
	limiter, clock := newTestLimiter()

	assert.InDelta(t, 10.0, limiter.Peek(Key{Name: "account", Since: clock.Now().Add(-time.Hour)}).Tokens, 1e-9)
	assert.InDelta(t, 10.0, limiter.Peek(Key{Name: "account", Since: clock.Now().Add(-8 * time.Second), Start: 5}).Tokens, 1e-9,
		"never past the burst")
	assert.InDelta(t, 3.0, limiter.Peek(Key{Name: "account", Since: clock.Now().Add(time.Hour), Start: 3}).Tokens, 1e-9,
		"a start in the future earns nothing yet")
}

func TestAScaledNewcomerEarnsAtItsScale(t *testing.T) {
	limiter, clock := newTestLimiter()

	assert.InDelta(t, 20.0, limiter.Peek(Key{Name: "campus", Scale: 10, Since: clock.Now().Add(-2 * time.Second)}).Tokens, 1e-9)
}

func TestARefilledNewcomerIsKeptUntilItHasEarnedItsBurst(t *testing.T) {
	limiter, clock := newTestLimiter()
	newcomer := Key{Name: "account", Since: clock.Now()}

	filled, state := limiter.Fill(newcomer)
	require.True(t, filled, "a new bucket that is not full has room")
	assert.InDelta(t, 10.0, state.Tokens, 1e-9)

	clock.Advance(time.Second)
	limiter.sweep()
	require.Contains(t, limiter.buckets, "account", "forgotten, it would come back with one token")

	clock.Advance(9 * time.Second)
	limiter.sweep()
	require.NotContains(t, limiter.buckets, "account")
	assert.InDelta(t, 10.0, limiter.Peek(newcomer).Tokens, 1e-9)
}
