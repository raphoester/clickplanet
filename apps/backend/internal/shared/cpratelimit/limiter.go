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

// Capacity is the burst this config refills to, defaults applied.
func (c Config) Capacity() int {
	return c.withDefaults().Burst
}

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

func New(config Config, clock cptime.Clock) *Limiter {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Limiter{
		config:  config.withDefaults(),
		clock:   clock,
		buckets: make(map[string]*bucket),
	}
}

type Limiter struct {
	config Config
	clock  cptime.Clock

	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time

	// A boost multiplies both the refill rate and the ceiling, until it lapses.
	// One means no boost, which is what every bucket that nothing ever boosted
	// holds — so a limiter nobody calls Boost on behaves exactly as it did
	// before boosting existed.
	multiplier float64
	boostUntil time.Time
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

// TakeN spends n tokens at once, or none: a click that costs three is refused on two.
func (l *Limiter) TakeN(key string, n int) (bool, State) {
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		b = newBucket(float64(l.config.Burst), now)
		l.buckets[key] = b
	}

	l.refill(b, now)

	if b.tokens < float64(n) {
		return false, l.state(b)
	}

	b.tokens -= float64(n)
	return true, l.state(b)
}

// Peek reports the state without spending anything. An unknown key is a full
// bucket and stays unknown: reading an allowance must not be a way to make the
// limiter remember an address that never clicked.
func (l *Limiter) Peek(key string) State {
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		return l.state(newBucket(float64(l.config.Burst), now))
	}

	l.refill(b, now)

	return l.state(b)
}

// Boost multiplies what a key may spend, and how fast it gets it back, until
// `until`.
//
// The tokens already in the bucket are left where they are: a boost widens the
// allowance and the rate, it does not hand out a full one. It is additive to the
// package in the strictest sense — nothing that never calls this can tell it
// exists — which matters because the same limiter type throttles chat and
// session mints, and neither has any business being boosted.
func (l *Limiter) Boost(key string, multiplier float64, until time.Time) State {
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		b = newBucket(float64(l.config.Burst), now)
		l.buckets[key] = b
	}

	l.refill(b, now)

	if multiplier > 1 && until.After(now) {
		b.multiplier = multiplier
		b.boostUntil = until
	}

	return l.state(b)
}

func newBucket(tokens float64, now time.Time) *bucket {
	return &bucket{tokens: tokens, last: now, multiplier: 1}
}

// state reports the reading together with the policy it refills under — which,
// while a boost runs, is the boosted one. A client replays that arithmetic to
// draw the allowance, so a boosted bucket widens the meter on screen with
// nothing on the client to change.
func (l *Limiter) state(b *bucket) State {
	return State{
		Tokens:    b.tokens,
		Capacity:  int(l.capacity(b)),
		PerSecond: l.config.PerSecond * b.multiplier,
	}
}

func (l *Limiter) capacity(b *bucket) float64 {
	return float64(l.config.Burst) * b.multiplier
}

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

		// A boosted bucket is not the same as a fresh one, so forgetting it
		// would quietly end the boost early. refill has already dropped the
		// multiplier of any boost that has lapsed, so this only holds the ones
		// still running.
		if b.multiplier > 1 {
			continue
		}

		if b.tokens >= float64(l.config.Burst) {
			delete(l.buckets, key)
		}
	}
}

// refill grants back what the elapsed time is worth, at the rate in force over
// it.
//
// The interval is **split at the moment a boost lapses**: an interval that
// straddles the end would otherwise be paid entirely at one rate or the other,
// over-granting a caller that went quiet across it. When the boost does end the
// tokens are clamped back to the plain burst, because the ceiling came down
// with it — a bucket left holding thirty under a burst of ten would spend the
// difference long after the minute was up.
func (l *Limiter) refill(b *bucket, now time.Time) {
	if !now.After(b.last) {
		return
	}

	if b.multiplier > 1 && b.boostUntil.After(b.last) {
		until := b.boostUntil
		if until.After(now) {
			until = now
		}

		b.tokens = math.Min(b.tokens+until.Sub(b.last).Seconds()*l.config.PerSecond*b.multiplier, l.capacity(b))
		b.last = until
	}

	if b.multiplier > 1 && !b.boostUntil.After(b.last) {
		b.multiplier = 1
		b.boostUntil = time.Time{}
		b.tokens = math.Min(b.tokens, float64(l.config.Burst))
	}

	if !now.After(b.last) {
		return
	}

	b.tokens = math.Min(b.tokens+now.Sub(b.last).Seconds()*l.config.PerSecond*b.multiplier, l.capacity(b))
	b.last = now
}
