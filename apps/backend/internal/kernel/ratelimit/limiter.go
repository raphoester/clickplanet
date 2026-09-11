package ratelimit

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
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

func New(config Config, timeProvider xtime.Provider) *Limiter {
	if timeProvider == nil {
		timeProvider = xtime.ActualProvider{}
	}

	return &Limiter{
		config:       config.withDefaults(),
		timeProvider: timeProvider,
		buckets:      make(map[string]*bucket),
	}
}

type Limiter struct {
	config       Config
	timeProvider xtime.Provider

	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
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
	now := l.timeProvider.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: float64(l.config.Burst), last: now}
		l.buckets[key] = b
	}

	l.refill(b, now)

	if b.tokens < 1 {
		return false, l.state(b.tokens)
	}

	b.tokens--
	return true, l.state(b.tokens)
}

// Peek reports the state without spending anything. An unknown key is a full
// bucket and stays unknown: reading an allowance must not be a way to make the
// limiter remember an address that never clicked.
func (l *Limiter) Peek(key string) State {
	now := l.timeProvider.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		return l.state(float64(l.config.Burst))
	}

	l.refill(b, now)

	return l.state(b.tokens)
}

func (l *Limiter) state(tokens float64) State {
	return State{
		Tokens:    tokens,
		Capacity:  l.config.Burst,
		PerSecond: l.config.PerSecond,
	}
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
	now := l.timeProvider.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	for key, b := range l.buckets {
		l.refill(b, now)
		if b.tokens >= float64(l.config.Burst) {
			delete(l.buckets, key)
		}
	}
}

func (l *Limiter) refill(b *bucket, now time.Time) {
	elapsed := now.Sub(b.last).Seconds()
	if elapsed <= 0 {
		return
	}

	b.tokens = math.Min(b.tokens+elapsed*l.config.PerSecond, float64(l.config.Burst))
	b.last = now
}
