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

	scale float64

	pace float64

	since time.Time
	start float64
}

type State struct {
	Tokens float64

	Capacity int

	PerSecond float64
}

func (l *Limiter) Take(key string) (bool, State) {
	return l.TakeN(key, 1)
}

func (l *Limiter) TakeN(key string, n float64) (bool, State) {
	allowed, states := l.TakeAll(n, Key{Name: key})
	return allowed, states[0]
}

type Key struct {
	Name  string
	Scale float64
	Pace  float64
	Since time.Time
	Start float64
}

func (l *Limiter) TakeAll(n float64, keys ...Key) (bool, []State) {
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	buckets := make([]*bucket, len(keys))
	allowed := true

	for i, key := range keys {
		b, ok := l.buckets[key.Name]
		if !ok {
			b = l.newBucket(key, now)
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

func (l *Limiter) Peek(key Key) State {
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key.Name]
	if !ok {
		// Not stored: reading an allowance must not make the limiter remember a caller.
		return l.state(l.newBucket(key, now))
	}

	l.refill(b, now)

	return l.state(b)
}

func (l *Limiter) Fill(key Key) (bool, State) {
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key.Name]
	if !ok {
		b = l.newBucket(key, now)
		if b.tokens >= l.capacity(b) {
			return false, l.state(b)
		}
		l.buckets[key.Name] = b
	}

	l.refill(b, now)
	if b.tokens >= l.capacity(b) {
		return false, l.state(b)
	}

	b.tokens = l.capacity(b)

	return true, l.state(b)
}

func (l *Limiter) newBucket(key Key, now time.Time) *bucket {
	scale := max(key.Scale, 1)
	b := &bucket{last: now, scale: scale, pace: 1, since: key.Since, start: key.Start}
	b.tokens = l.earned(b, now)

	return b
}

func (l *Limiter) earned(b *bucket, now time.Time) float64 {
	if b.since.IsZero() {
		return l.capacity(b)
	}

	return math.Min(max(b.start, 0)+max(now.Sub(b.since).Seconds(), 0)*l.config.PerSecond*b.scale, l.capacity(b))
}

func (l *Limiter) state(b *bucket) State {
	return State{
		Tokens:    b.tokens,
		Capacity:  int(l.capacity(b)),
		PerSecond: l.rate(b),
	}
}

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

		// Keep it until a new bucket would hold as much, or recreating it would lose tokens.
		if b.tokens >= l.capacity(b) && l.earned(b, now) >= l.capacity(b) {
			delete(l.buckets, key)
		}
	}
}

func (l *Limiter) refill(b *bucket, now time.Time) {
	if !now.After(b.last) {
		return
	}

	b.tokens = math.Min(b.tokens+now.Sub(b.last).Seconds()*l.rate(b), l.capacity(b))
	b.last = now
}
