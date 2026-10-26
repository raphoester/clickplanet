package cpratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type ruling struct {
	clock  *cptime.FixedClock
	policy Policy
}

func (r *ruling) read() Policy { return r.policy }

func (r *ruling) enforce(policy Policy) {
	policy.Since = r.clock.Now()
	r.policy = policy
}

var plain = Policy{PerSecond: 1, Burst: 10, Tempo: 1}

func newRuledLimiter() (*Limiter, *ruling) {
	clock := cptime.NewFixedClock(epoch)
	rules := &ruling{clock: clock}
	rules.enforce(plain)

	return NewRuled("test", rules.read, time.Minute, clock), rules
}

func withTempo(tempo float64) Policy {
	policy := plain
	policy.Tempo = tempo
	return policy
}

func TestATempoSpeedsUpABucketFromThenOnAndNotBefore(t *testing.T) {
	limiter, rules := newRuledLimiter()
	key := Key{Name: "account:alice", Scale: 1}
	allowed, _ := limiter.TakeAll(10, key)
	require.True(t, allowed)

	rules.clock.Advance(2 * time.Second)
	rules.enforce(withTempo(3))
	rules.clock.Advance(time.Second)

	state := limiter.Peek(key)
	assert.InDelta(t, 2+3, state.Tokens, 1e-9, "two seconds at 1, then one at 3")
	assert.InDelta(t, 3.0, state.PerSecond, 1e-9)
}

func TestATempoMultipliesTheScaleAndThePaceOfEveryBucket(t *testing.T) {
	limiter, rules := newRuledLimiter()
	rules.enforce(withTempo(3))

	_, states := limiter.TakeAll(1,
		Key{Name: "account:alice", Scale: 1, Pace: 2},
		Key{Name: "scope:1.2.3.4", Scale: 10})

	assert.InDelta(t, 6.0, states[0].PerSecond, 1e-9)
	assert.InDelta(t, 30.0, states[1].PerSecond, 1e-9)
	assert.Equal(t, 10, states[0].Capacity, "the bank keeps its size")
}

func TestAPeekOfABucketNeverMadeReadsTheTempo(t *testing.T) {
	limiter, rules := newRuledLimiter()
	rules.enforce(withTempo(3))

	state := limiter.Peek(Key{Name: "account:bob", Scale: 1})

	assert.InDelta(t, 3.0, state.PerSecond, 1e-9)
	assert.InDelta(t, 10.0, state.Tokens, 1e-9)
}

func TestTheTempoGoesBackDown(t *testing.T) {
	limiter, rules := newRuledLimiter()
	key := Key{Name: "account:alice", Scale: 1}
	rules.enforce(withTempo(3))
	allowed, _ := limiter.TakeAll(10, key)
	require.True(t, allowed)

	rules.clock.Advance(time.Second)
	rules.enforce(plain)
	rules.clock.Advance(time.Second)

	assert.InDelta(t, 3+1, limiter.Peek(key).Tokens, 1e-9)
}

func TestARateIsRefilledAtTheOldRateUpToTheChangeEvenWhenNothingWasAskedSince(t *testing.T) {
	limiter, rules := newRuledLimiter()
	key := Key{Name: "account:alice", Scale: 1}
	allowed, _ := limiter.TakeAll(10, key)
	require.True(t, allowed)

	rules.clock.Advance(2 * time.Second)
	quicker := plain
	quicker.PerSecond = 2
	rules.enforce(quicker)
	rules.clock.Advance(3 * time.Second)

	assert.InDelta(t, 2*1+3*2, limiter.Peek(key).Tokens, 1e-9, "the change is read at the peek, three seconds after it")
}

func TestASmallerBurstCutsEveryBankToIt(t *testing.T) {
	limiter, rules := newRuledLimiter()
	key := Key{Name: "scope:1.2.3.4", Scale: 2}
	_, _ = limiter.TakeAll(1, key)

	smaller := plain
	smaller.Burst = 5
	rules.enforce(smaller)

	state := limiter.Peek(key)
	assert.InDelta(t, 10.0, state.Tokens, 1e-9, "19 tokens held, 10 fit")
	assert.Equal(t, 10, state.Capacity)
}

func TestABiggerBurstIsEarnedNotHandedOver(t *testing.T) {
	limiter, rules := newRuledLimiter()
	key := Key{Name: "account:alice", Scale: 1}
	_, _ = limiter.TakeAll(1, key)

	bigger := plain
	bigger.Burst = 20
	rules.enforce(bigger)
	rules.clock.Advance(4 * time.Second)

	state := limiter.Peek(key)
	assert.InDelta(t, 9+4, state.Tokens, 1e-9)
	assert.Equal(t, 20, state.Capacity)
}

func TestABucketFollowsTheScaleOfItsKeyFromATakeOn(t *testing.T) {
	limiter, rules := newRuledLimiter()
	_, _ = limiter.TakeAll(4, Key{Name: "scope:1.2.3.4", Scale: 10})

	rules.clock.Advance(time.Second)
	_, states := limiter.TakeAll(1, Key{Name: "scope:1.2.3.4", Scale: 4})

	assert.Equal(t, 40, states[0].Capacity)
	assert.InDelta(t, 40-1, states[0].Tokens, 1e-9, "96 earned at the old scale, cut to the new size, then one spent")
	assert.InDelta(t, 4.0, states[0].PerSecond, 1e-9)
}
