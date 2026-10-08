package shadowban

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Config struct {
	Enforce bool

	BanDurations []time.Duration

	ReflagInterval time.Duration
}

const defaultReflagInterval = 5 * time.Minute

func (c Config) withDefaults() Config {
	if len(c.BanDurations) == 0 {
		c.BanDurations = []time.Duration{24 * time.Hour, 7 * 24 * time.Hour, 3 * 365 * 24 * time.Hour}
	}
	if c.ReflagInterval <= 0 {
		c.ReflagInterval = defaultReflagInterval
	}
	return c
}

type Sentence struct {
	Flags   int
	Offence int
	Until   time.Time
}

func New(config Config, clock cptime.Clock, store Store) *Banner {
	return &Banner{config: config.withDefaults(), clock: clock, store: store}
}

type Banner struct {
	config Config
	clock  cptime.Clock
	store  Store
}

func (r Record) running(now time.Time) bool {
	return now.Before(r.Until)
}

func (r Record) sentence() Sentence {
	return Sentence{Flags: r.Flags, Offence: r.Offences, Until: r.Until}
}

func (r Record) extendedTo(until time.Time) Record {
	if until.After(r.Until) {
		r.Until = until
	}
	return r
}

func (b *Banner) flagged(record Record, now time.Time) Record {
	if !record.running(now) {
		record.Offences++
	}
	record.Flags++
	record.NextFlagAt = now.Add(b.config.ReflagInterval)

	return record.extendedTo(now.Add(b.duration(record.Offences)))
}

func (b *Banner) banned(record Record, now time.Time, duration time.Duration) Record {
	if !record.running(now) {
		record.Offences++
	}
	if duration <= 0 {
		duration = b.duration(record.Offences)
	}

	return record.extendedTo(now.Add(duration))
}

func lifted(record Record, now time.Time) Record {
	record.Offences = max(record.Offences-1, 0)
	record.Until = now
	return record
}

func (b *Banner) Flag(ctx context.Context, scope string) (Sentence, bool, error) {
	if scope == "" {
		return Sentence{}, false, nil
	}

	now := b.clock.Now()

	var (
		sentence Sentence
		accepted bool
	)

	if err := b.store.Change(ctx, scope, func(record Record, found bool) (Record, bool) {
		if found && now.Before(record.NextFlagAt) {
			sentence = record.sentence()
			return record, false
		}

		record = b.flagged(record, now)
		sentence, accepted = record.sentence(), true
		return record, true
	}); err != nil {
		return Sentence{}, false, fmt.Errorf("failed to flag %q: %w", scope, err)
	}

	return sentence, accepted, nil
}

func (b *Banner) Ban(ctx context.Context, scope string, duration time.Duration) (Sentence, error) {
	if scope == "" {
		return Sentence{}, nil
	}

	now := b.clock.Now()

	var sentence Sentence
	if err := b.store.Change(ctx, scope, func(record Record, _ bool) (Record, bool) {
		record = b.banned(record, now, duration)
		sentence = record.sentence()
		return record, true
	}); err != nil {
		return Sentence{}, fmt.Errorf("failed to ban %q: %w", scope, err)
	}

	return sentence, nil
}

func (b *Banner) Unban(ctx context.Context, scope string) error {
	if scope == "" {
		return nil
	}

	now := b.clock.Now()

	if err := b.store.Change(ctx, scope, func(record Record, found bool) (Record, bool) {
		if !found || !record.running(now) {
			return record, false
		}
		return lifted(record, now), true
	}); err != nil {
		return fmt.Errorf("failed to unban %q: %w", scope, err)
	}

	return nil
}

func (b *Banner) Sentence(ctx context.Context, scope string) (Sentence, bool, error) {
	if scope == "" {
		return Sentence{}, false, nil
	}

	record, found, err := b.store.Record(ctx, scope)
	if err != nil {
		return Sentence{}, false, fmt.Errorf("failed to read the ban on %q: %w", scope, err)
	}
	if !found || !record.running(b.clock.Now()) {
		return Sentence{}, false, nil
	}

	return record.sentence(), true, nil
}

func (b *Banner) duration(offence int) time.Duration {
	ladder := b.config.BanDurations
	if offence > len(ladder) {
		return ladder[len(ladder)-1]
	}
	return ladder[offence-1]
}

func (b *Banner) Banned(ctx context.Context, scope string) (bool, error) {
	if !b.config.Enforce {
		return false, nil
	}

	_, running, err := b.Sentence(ctx, scope)
	return running, err
}

func (b *Banner) Flagged(ctx context.Context) (int, error) {
	running, err := b.store.Running(ctx, b.clock.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to count the running bans: %w", err)
	}
	return running, nil
}

func (b *Banner) Enforcing() bool { return b.config.Enforce }
