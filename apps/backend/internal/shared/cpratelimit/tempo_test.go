package cpratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestATempoSpeedsUpABucketFromThenOnAndNotBefore(t *testing.T) {
	limiter, clock := newTestLimiter()
	key := Key{Name: "account:alice", Scale: 1}
	allowed, _ := limiter.TakeAll(10, key)
	require.True(t, allowed)

	clock.Advance(2 * time.Second)
	limiter.SetTempo(3)
	clock.Advance(time.Second)

	state := limiter.Peek(key)
	assert.InDelta(t, 2+3, state.Tokens, 1e-9, "two seconds at 1, then one at 3")
	assert.InDelta(t, 3.0, state.PerSecond, 1e-9)
}

func TestATempoMultipliesTheScaleAndThePaceOfEveryBucket(t *testing.T) {
	limiter, _ := newTestLimiter()
	limiter.SetTempo(3)

	_, states := limiter.TakeAll(1,
		Key{Name: "account:alice", Scale: 1, Pace: 2},
		Key{Name: "scope:1.2.3.4", Scale: 10})

	assert.InDelta(t, 6.0, states[0].PerSecond, 1e-9)
	assert.InDelta(t, 30.0, states[1].PerSecond, 1e-9)
	assert.Equal(t, 10, states[0].Capacity, "the bank keeps its size")
}

func TestAPeekOfABucketNeverMadeReadsTheTempo(t *testing.T) {
	limiter, _ := newTestLimiter()
	limiter.SetTempo(3)

	state := limiter.Peek(Key{Name: "account:bob", Scale: 1})

	assert.InDelta(t, 3.0, state.PerSecond, 1e-9)
	assert.InDelta(t, 10.0, state.Tokens, 1e-9)
}

func TestTheTempoGoesBackDown(t *testing.T) {
	limiter, clock := newTestLimiter()
	key := Key{Name: "account:alice", Scale: 1}
	limiter.SetTempo(3)
	allowed, _ := limiter.TakeAll(10, key)
	require.True(t, allowed)

	clock.Advance(time.Second)
	limiter.SetTempo(1)
	clock.Advance(time.Second)

	assert.InDelta(t, 3+1, limiter.Peek(key).Tokens, 1e-9)
}
