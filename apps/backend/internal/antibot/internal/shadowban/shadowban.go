// Package shadowban is the consequence, and nothing else: scopes, a clock and how long to ban for.
package shadowban

import (
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

	// LegacyStatePath is the bans file from before postgres, imported once into an empty table.
	LegacyStatePath string
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

func New(config Config, clock cptime.Clock, persistence Persistence, onStateError func(error)) *Banner {
	return &Banner{
		config:       config.withDefaults(),
		clock:        clock,
		persistence:  persistence,
		onStateError: onStateError,
		bans:         make(map[string]*ban),
		dirty:        make(map[string]struct{}),
	}
}

type Banner struct {
	config       Config
	clock        cptime.Clock
	persistence  Persistence
	onStateError func(error)

	mu       sync.Mutex
	bans     map[string]*ban
	dirty    map[string]struct{}
	imported string // the legacy file loaded, renamed after the first flush
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

	b.dirty[scope] = struct{}{}

	return record.sentence(), true
}

// Ban is a ban an operator decided on. It counts as an offence like a flag does; a zero duration takes the ladder's.
func (b *Banner) Ban(scope string, duration time.Duration) Sentence {
	if scope == "" {
		return Sentence{}
	}

	now := b.clock.Now()

	b.mu.Lock()
	defer b.mu.Unlock()

	record, ok := b.bans[scope]
	if !ok {
		record = &ban{}
		b.bans[scope] = record
	}

	if !record.running(now) {
		record.offences++
	}
	if duration <= 0 {
		duration = b.duration(record.offences)
	}
	if until := now.Add(duration); until.After(record.until) {
		record.until = until
	}

	b.dirty[scope] = struct{}{}

	return record.sentence()
}

// Sentence is the scope's ban, if one is running.
func (b *Banner) Sentence(scope string) (Sentence, bool) {
	now := b.clock.Now()

	b.mu.Lock()
	defer b.mu.Unlock()

	record, ok := b.bans[scope]
	if !ok || !record.running(now) {
		return Sentence{}, false
	}

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
