// Package ratelimit implements a token bucket per key — a source IP, in
// practice. Everything lives in this process, like the tile map it protects:
// there is a single API instance, so a shared counter would buy nothing.
package ratelimit

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

// Config describes one bucket. PerSecond is the allowance a caller gets back
// steadily; Burst is what a caller arriving after a quiet spell may spend at
// once, which is what keeps a normal player's click flurry from being refused.
type Config struct {
	// PerSecond is how fast a bucket refills. Defaults to 1.
	PerSecond float64

	// Burst is the bucket's capacity: the most a caller can spend before the
	// refill rate is all that is left to them. Defaults to 10.
	Burst int

	// SweepInterval is how often Run forgets the buckets that have refilled
	// completely. Defaults to 1m.
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

// New builds a limiter holding no buckets: a key is only remembered once it
// has spent something.
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

	// mu guards the bucket map and every bucket in it. The critical section is
	// a map lookup and a subtraction, so one mutex is enough at click rates.
	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

// Allow spends one token from key's bucket and reports whether there was one
// to spend. A key seen for the first time starts with a full bucket.
func (l *Limiter) Allow(key string) bool {
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
		return false
	}

	b.tokens--
	return true
}

// Run forgets idle buckets every SweepInterval until ctx is done. A bucket
// that has refilled to capacity holds exactly what a bucket created on the
// spot would, so dropping it costs its owner nothing — and without that the
// map would keep one entry per address that ever clicked.
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

// refill credits the time since the bucket was last touched, capped at the
// bucket's capacity. A clock that jumps backwards credits nothing rather than
// taking tokens away.
func (l *Limiter) refill(b *bucket, now time.Time) {
	elapsed := now.Sub(b.last).Seconds()
	if elapsed <= 0 {
		return
	}

	b.tokens = math.Min(b.tokens+elapsed*l.config.PerSecond, float64(l.config.Burst))
	b.last = now
}
