package shadowban

import (
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Config struct {
	Enforce bool

	BanDurations []time.Duration

	ReflagInterval time.Duration
	SaveInterval   time.Duration
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
		dirty:        cpcolls.NewSet[string](),
	}
}

type Banner struct {
	config       Config
	clock        cptime.Clock
	persistence  Persistence
	onStateError func(error)

	mu    sync.Mutex
	bans  map[string]*ban
	dirty *cpcolls.Set[string]
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

	if !record.running(now) {
		record.offences++
	}
	record.flags++
	record.nextFlagAt = now.Add(b.config.ReflagInterval)

	if until := now.Add(b.duration(record.offences)); until.After(record.until) {
		record.until = until
	}

	b.dirty.Add(scope)

	return record.sentence(), true
}

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

	b.dirty.Add(scope)

	return record.sentence()
}

func (b *Banner) Unban(scope string) {
	now := b.clock.Now()

	b.mu.Lock()
	defer b.mu.Unlock()

	record, ok := b.bans[scope]
	if !ok || !record.running(now) {
		return
	}

	record.offences = max(record.offences-1, 0)
	record.until = now

	b.dirty.Add(scope)
}

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
