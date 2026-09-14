// Package shadowban is the consequence, and nothing else: scopes, a clock and how long to ban for.
package shadowban

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Config struct {
	Enforce bool

	// The Nth offence bans for the Nth entry; past the end the last one repeats.
	BanDurations []time.Duration

	ReflagInterval time.Duration
	SaveInterval   time.Duration

	StatePath string
}

const (
	defaultReflagInterval = 5 * time.Minute
	defaultSaveInterval   = time.Minute
)

func (c Config) withDefaults() Config {
	if len(c.BanDurations) == 0 {
		c.BanDurations = []time.Duration{24 * time.Hour, 7 * 24 * time.Hour, 3 * 365 * 24 * time.Hour}
	}
	if c.ReflagInterval <= 0 {
		c.ReflagInterval = defaultReflagInterval
	}
	if c.SaveInterval <= 0 {
		c.SaveInterval = defaultSaveInterval
	}
	return c
}

type Sentence struct {
	Flags   int
	Offence int
	Until   time.Time
}

func New(config Config, clock cptime.Clock, onStateError func(error)) *Banner {
	if clock == nil {
		clock = cptime.SystemClock{}
	}
	if onStateError == nil {
		onStateError = func(error) {}
	}

	b := &Banner{
		config:       config.withDefaults(),
		clock:        clock,
		onStateError: onStateError,
		bans:         make(map[string]*ban),
	}

	if err := b.restore(); err != nil {
		onStateError(err)
	}

	return b
}

type Banner struct {
	config       Config
	clock        cptime.Clock
	onStateError func(error)

	mu    sync.Mutex
	bans  map[string]*ban
	dirty bool
}

type ban struct {
	flags      int
	offences   int
	until      time.Time
	nextFlagAt time.Time
}

func (r *ban) running(now time.Time) bool {
	return now.Before(r.until)
}

func (r *ban) sentence() Sentence {
	return Sentence{Flags: r.flags, Offence: r.offences, Until: r.until}
}

func (b *Banner) Flag(scope string) (Sentence, bool) {
	if scope == "" {
		return Sentence{}, false
	}

	now := b.clock.Now()

	b.mu.Lock()
	defer b.mu.Unlock()

	record, ok := b.bans[scope]
	if !ok {
		record = &ban{}
		b.bans[scope] = record
	}

	if ok && now.Before(record.nextFlagAt) {
		return record.sentence(), false
	}

	// A flag on a running ban extends it; only a ban starting fresh is a new offence.
	if !record.running(now) {
		record.offences++
	}
	record.flags++
	record.nextFlagAt = now.Add(b.config.ReflagInterval)

	if until := now.Add(b.duration(record.offences)); until.After(record.until) {
		record.until = until
	}

	b.dirty = true

	return record.sentence(), true
}

func (b *Banner) duration(offence int) time.Duration {
	ladder := b.config.BanDurations
	if offence > len(ladder) {
		return ladder[len(ladder)-1]
	}
	return ladder[offence-1]
}

func (b *Banner) Banned(scope string) bool {
	if !b.config.Enforce {
		return false
	}

	now := b.clock.Now()

	b.mu.Lock()
	defer b.mu.Unlock()

	record, ok := b.bans[scope]
	return ok && record.running(now)
}

func (b *Banner) Flagged() int {
	now := b.clock.Now()

	b.mu.Lock()
	defer b.mu.Unlock()

	var count int
	for _, record := range b.bans {
		if record.running(now) {
			count++
		}
	}

	return count
}

func (b *Banner) Enforcing() bool { return b.config.Enforce }

func (b *Banner) Run(ctx context.Context) {
	ticker := time.NewTicker(b.config.SaveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.saveIfDirty()
		case <-ctx.Done():
			b.saveIfDirty()
			return
		}
	}
}
