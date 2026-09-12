// Package shadowban is the consequence, and nothing else. It knows about
// scopes, a clock and a ban duration; it knows nothing about bots, tiles or
// what earned the ban. Whoever decides passes a scope in.
package shadowban

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/xtime"
)

type Config struct {
	// Enforce off records and counts bans without dropping anything. It is the
	// mode to deploy in: what would have been dropped is visible in the log and
	// in shadowban_flagged, and no player pays for a bound that is still wrong.
	Enforce bool

	// BanDuration is how long one flag silences a caller.
	BanDuration time.Duration

	// ReflagInterval is how soon a caller already serving a ban can be flagged
	// again. Without it a caller that keeps going is flagged on every click and
	// the log says nothing; with it, flags=6 on a line means six separate times
	// the caller was judged and read the same way.
	ReflagInterval time.Duration

	SweepInterval time.Duration
}

const (
	defaultBanDuration = time.Hour
	// Sized to match a watchdog's track window, so consecutive flags rest on
	// evidence the previous one never saw.
	defaultReflagInterval = 5 * time.Minute
	defaultSweepInterval  = time.Minute
)

func (c Config) withDefaults() Config {
	if c.BanDuration <= 0 {
		c.BanDuration = defaultBanDuration
	}
	if c.ReflagInterval <= 0 {
		c.ReflagInterval = defaultReflagInterval
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

func New(config Config, timeProvider xtime.Provider) *Banner {
	if timeProvider == nil {
		timeProvider = xtime.ActualProvider{}
	}

	return &Banner{
		config:       config.withDefaults(),
		timeProvider: timeProvider,
		bans:         make(map[string]*ban),
	}
}

type Banner struct {
	config       Config
	timeProvider xtime.Provider

	mu   sync.Mutex
	bans map[string]*ban
}

type ban struct {
	flags      int
	until      time.Time
	nextFlagAt time.Time
}

// Flag bans a scope. It returns how many times this scope has been flagged, and
// false when the previous flag is still inside ReflagInterval — the ban is
// already running and there is nothing new to say about it.
func (b *Banner) Flag(scope string) (flags int, accepted bool) {
	if scope == "" {
		return 0, false
	}

	now := b.timeProvider.Now()

	b.mu.Lock()
	defer b.mu.Unlock()

	record, ok := b.bans[scope]
	if !ok {
		record = &ban{}
		b.bans[scope] = record
	}

	if ok && now.Before(record.nextFlagAt) {
		return record.flags, false
	}

	record.flags++
	record.until = now.Add(b.config.BanDuration)
	record.nextFlagAt = now.Add(b.config.ReflagInterval)

	return record.flags, true
}

// Banned is whether this scope's clicks should be dropped. It is false while
// Enforce is off, however many flags the scope has.
func (b *Banner) Banned(scope string) bool {
	if !b.config.Enforce {
		return false
	}

	now := b.timeProvider.Now()

	b.mu.Lock()
	defer b.mu.Unlock()

	record, ok := b.bans[scope]
	return ok && now.Before(record.until)
}

// Flagged counts the bans currently running, whether or not Enforce is on, so
// the gauge answers "what would this drop" before anything is dropped.
func (b *Banner) Flagged() int {
	now := b.timeProvider.Now()

	b.mu.Lock()
	defer b.mu.Unlock()

	var count int
	for _, record := range b.bans {
		if now.Before(record.until) {
			count++
		}
	}

	return count
}

func (b *Banner) Enforcing() bool { return b.config.Enforce }

func (b *Banner) Run(ctx context.Context) {
	ticker := time.NewTicker(b.config.SweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.sweep()
		case <-ctx.Done():
			return
		}
	}
}

func (b *Banner) sweep() {
	now := b.timeProvider.Now()

	b.mu.Lock()
	defer b.mu.Unlock()

	for scope, record := range b.bans {
		// The flag count is the point of keeping a lapsed ban around, so a
		// scope is only forgotten once it could be flagged from scratch without
		// losing anything a log line would have shown.
		if now.Before(record.until) || now.Before(record.nextFlagAt) {
			continue
		}
		delete(b.bans, scope)
	}
}
