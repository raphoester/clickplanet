package cpratelimit

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Config struct {
	PerSecond float64

	Burst int

	SweepInterval time.Duration
}

const (
	defaultPerSecond     = 1
	defaultBurst         = 10
	defaultSweepInterval = time.Minute
)

func (c Config) withDefaults() Config {
	if c.PerSecond <= 0 {
		c.PerSecond = defaultPerSecond
	}
	if c.Burst <= 0 {
		c.Burst = defaultBurst
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

// New takes a name because one process runs several limiters.
func New(name string, config Config, clock cptime.Clock) *Limiter {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Limiter{
		name:    name,
		config:  config.withDefaults(),
		clock:   clock,
		buckets: make(map[string]*bucket),
	}
}

type Limiter struct {
	name   string
	config Config
	clock  cptime.Clock

	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time

	// Scale multiplies the burst and the rate for good, set when the bucket is made.
	scale float64

	// Pace multiplies the rate, from the last take that set it on. One is the plain rate.
	pace float64
}

// State is what a bucket holds, together with the policy it refills under. The
// policy travels with the reading because a caller that shows the allowance to
// a human replays the refill itself between two readings, instead of asking
// again every frame.
type State struct {
	// Tokens left, fractional, so a caller can show the next one arriving.
	Tokens float64

	// The most a bucket can bank: the burst.
	Capacity int

	// Tokens granted back per second.
	PerSecond float64
}

// Take spends a token when there is one, and reports what the bucket holds
// afterwards. The state comes back either way: a refused caller is the one
// most interested in how long the wait is.
func (l *Limiter) Take(key string) (bool, State) {
	return l.TakeN(key, 1)
}

// TakeN spends n tokens at once, or none.
func (l *Limiter) TakeN(key string, n float64) (bool, State) {
	allowed, states := l.TakeAll(n, Key{Name: key})
	return allowed, states[0]
}

// Key names a bucket, and the scale its burst and rate are multiplied by. A scale under one is one.
//
// Pace, when set, multiplies the bucket's rate from this take on, and leaves its burst alone: a take
// does not reprice the time already past, which was refilled at the rate in force over it. Peek
// ignores it. Zero leaves the bucket's pace as it is.
type Key struct {
	Name  string
	Scale float64
	Pace  float64
}

// TakeAll spends n tokens from every bucket, or from none: one bucket refusing
// spends nothing from the others. The states come back in the order of the keys.
//
// A key's pace is set whether or not the take is allowed: it says what the caller is doing now,
// and a refused caller waits at that rate too.
func (l *Limiter) TakeAll(n float64, keys ...Key) (bool, []State) {
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	buckets := make([]*bucket, len(keys))
	allowed := true

	for i, key := range keys {
		b, ok := l.buckets[key.Name]
		if !ok {
			b = l.newBucket(key.Scale, now)
			l.buckets[key.Name] = b
		}

		l.refill(b, now)
		if key.Pace > 0 {
			b.pace = key.Pace
		}

		buckets[i] = b
		allowed = allowed && b.tokens >= n
	}

	states := make([]State, len(buckets))
	for i, b := range buckets {
		if allowed {
			b.tokens -= n
		}
		states[i] = l.state(b)
	}

	return allowed, states
}

// Peek reports the state without spending anything. An unknown key is a full
// bucket and stays unknown: reading an allowance must not be a way to make the
// limiter remember an address that never clicked.
func (l *Limiter) Peek(key Key) State {
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key.Name]
	if !ok {
		return l.state(l.newBucket(key.Scale, now))
	}

	l.refill(b, now)

	return l.state(b)
}

// Fill tops a key's bucket up to its capacity, and reports whether there was room: a full bucket is
// left alone and answers false, so a caller does not spend something on nothing. An unknown key is a full
// bucket. It is additive to the package: nothing that never calls it can tell it exists, which matters
// because the same limiter type throttles chat and session mints.
func (l *Limiter) Fill(key Key) (bool, State) {
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key.Name]
	if !ok {
		return false, l.state(l.newBucket(key.Scale, now))
	}

	l.refill(b, now)
	if b.tokens >= l.capacity(b) {
		return false, l.state(b)
	}

	b.tokens = l.capacity(b)

	return true, l.state(b)
}

func (l *Limiter) newBucket(scale float64, now time.Time) *bucket {
	scale = max(scale, 1)
	return &bucket{tokens: float64(l.config.Burst) * scale, last: now, scale: scale, pace: 1}
}

// state reports the reading together with the policy it refills under. A client replays that arithmetic
// to draw the allowance.
func (l *Limiter) state(b *bucket) State {
	return State{
		Tokens:    b.tokens,
		Capacity:  int(l.capacity(b)),
		PerSecond: l.rate(b),
	}
}

// capacity is the burst, which nothing but the scale moves.
func (l *Limiter) capacity(b *bucket) float64 {
	return float64(l.config.Burst) * b.scale
}

func (l *Limiter) rate(b *bucket) float64 {
	return l.config.PerSecond * b.scale * b.pace
}

func (l *Limiter) Name() string { return l.name }

func (l *Limiter) Run(ctx context.Context) {
	ticker := time.NewTicker(l.config.SweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.sweep()
		case <-ctx.Done():
			return
		}
	}
}

func (l *Limiter) sweep() {
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	for key, b := range l.buckets {
		l.refill(b, now)

		if b.tokens >= l.capacity(b) {
			delete(l.buckets, key)
		}
	}
}

// refill grants back what the elapsed time is worth, at the rate in force over it.
func (l *Limiter) refill(b *bucket, now time.Time) {
	if !now.After(b.last) {
		return
	}

	b.tokens = math.Min(b.tokens+now.Sub(b.last).Seconds()*l.rate(b), l.capacity(b))
	b.last = now
}
