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
