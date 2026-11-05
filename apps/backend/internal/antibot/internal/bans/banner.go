package bans

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Rules struct {
	Enforce bool

	BanDurations []time.Duration

	ReflagInterval time.Duration
}

func (r Rules) Validate() error {
	errs := []error{detect.Positive("reflagInterval", r.ReflagInterval)}
	if len(r.BanDurations) == 0 {
		errs = append(errs, errors.New("banDurations is empty, and the first offence needs a ban"))
	}
	for i, duration := range r.BanDurations {
		errs = append(errs, detect.Positive(fmt.Sprintf("banDurations[%d]", i), duration))
	}
	return errors.Join(errs...)
}

type Sentence struct {
	Flags   int
	Offence int
	Until   time.Time
}

// What a ban asks of the game: the caller is refused until a time, as decided at another. Ended is admitted.
type Enforcer interface {
	Enforce(ctx context.Context, caller Caller, until, at time.Time) error
}

type Kind uint8

const (
	Scopes Kind = iota
	Accounts
)

func (k Kind) callerOf(key string) Caller {
	if k == Accounts {
		return Caller{Account: key}
	}
	return Caller{Scope: key}
}

func New(
	rules func() Rules,
	pardon time.Duration,
	clock cptime.Clock,
	kind Kind,
	store Store,
	transactioner cpatomic.Transactioner,
	enforcer Enforcer,
) *Banner {
	return &Banner{
		rules: rules, pardon: pardon, clock: clock,
		kind: kind, store: store, transactioner: transactioner, enforcer: enforcer,
	}
}

type Banner struct {
	rules  func() Rules
	pardon time.Duration
	clock  cptime.Clock

	kind          Kind
	store         Store
	transactioner cpatomic.Transactioner
	enforcer      Enforcer
}

func (r Record) running(now time.Time) bool {
	return now.Before(r.ExpiresAt)
}

func (r Record) liftable(now time.Time) bool {
	return r.running(now) || r.Offences > 0
}

func (p Pardon) running(now time.Time) bool {
	return now.Before(p.Until)
}

func (r Record) sentence() Sentence {
	return Sentence{Flags: r.Flags, Offence: r.Offences, Until: r.ExpiresAt}
}

func (r Record) extendedTo(until time.Time) Record {
	if until.After(r.ExpiresAt) {
		r.ExpiresAt = until
	}
	return r
}

func (b *Banner) flaggable(record Record, now time.Time) bool {
	return !record.Pardon.running(now) && !now.Before(record.LastFlaggedAt.Add(b.rules().ReflagInterval))
}

func (b *Banner) flagged(record Record, now time.Time) Record {
	if !record.running(now) {
		record.Offences++
	}
	record.Flags++
	record.LastFlaggedAt = now

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

func (b *Banner) lifted(record Record, now time.Time) Record {
	record.Pardon = Pardon{At: now, Lifted: record.sentence(), Until: now.Add(b.pardon)}
	record.Offences = max(record.Offences-1, 0)
	if record.running(now) {
		record.ExpiresAt = now
	}
	return record
}

func (b *Banner) Flag(ctx context.Context, scope string) (Sentence, bool, error) {
	if scope == "" {
		return Sentence{}, false, nil
	}

	var (
		sentence Sentence
		accepted bool
	)

	if err := b.change(ctx, scope, func(record Record, _ bool, now time.Time) (Record, bool) {
		if !b.flaggable(record, now) {
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

	var sentence Sentence
	if err := b.change(ctx, scope, func(record Record, _ bool, now time.Time) (Record, bool) {
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

	if err := b.change(ctx, scope, func(record Record, found bool, now time.Time) (Record, bool) {
		if !found || !record.liftable(now) {
			return record, false
		}
		return b.lifted(record, now), true
	}); err != nil {
		return fmt.Errorf("failed to unban %q: %w", scope, err)
	}

	return nil
}

// Each change kept tells the game, in its transaction: a ban kept and never refused is what an outbox prevents.
func (b *Banner) change(ctx context.Context, key string, change func(Record, bool, time.Time) (Record, bool)) error {
	return b.transactioner.InTx(ctx, func(ctx context.Context) error {
		var (
			kept Record
			told bool
			at   time.Time
		)
		if err := b.store.Change(ctx, key, func(record Record, found bool) (Record, bool) {
			at = b.clock.Now()
			kept, told = change(record, found, at)
			return kept, told
		}); err != nil {
			return err //nolint:wrapcheck // the caller names the change.
		}
		if !told {
			return nil
		}
		return b.enforcer.Enforce(ctx, b.kind.callerOf(key), b.enforced(kept, at), at) //nolint:wrapcheck // the caller names the change.
	})
}

func (b *Banner) enforced(record Record, at time.Time) time.Time {
	if !b.rules().Enforce {
		return at
	}
	return record.ExpiresAt
}

// At a boot: the bans running when it was deployed reach the game too.
func (b *Banner) Restate(ctx context.Context) error {
	at := b.clock.Now()

	return b.transactioner.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // each step names what failed.
		running, err := b.store.Running(ctx, at)
		if err != nil {
			return fmt.Errorf("failed to read the running bans: %w", err)
		}

		for _, record := range running {
			if err := b.enforcer.Enforce(ctx, b.kind.callerOf(record.Key), b.enforced(record, at), at); err != nil {
				return fmt.Errorf("failed to tell the ban on %q: %w", record.Key, err)
			}
		}
		return nil
	})
}

func (b *Banner) Sentence(ctx context.Context, scope string) (Sentence, bool, error) {
	return b.sentenceOf(ctx, scope, Record.running)
}

func (b *Banner) LastSentence(ctx context.Context, scope string) (Sentence, bool, error) {
	return b.sentenceOf(ctx, scope, Record.liftable)
}

func (b *Banner) sentenceOf(
	ctx context.Context,
	scope string,
	counted func(Record, time.Time) bool,
) (Sentence, bool, error) {
	if scope == "" {
		return Sentence{}, false, nil
	}

	record, found, err := b.store.Record(ctx, scope)
	if err != nil {
		return Sentence{}, false, fmt.Errorf("failed to read the ban on %q: %w", scope, err)
	}
	if !found || !counted(record, b.clock.Now()) {
		return Sentence{}, false, nil
	}

	return record.sentence(), true, nil
}

func (b *Banner) Pardon(ctx context.Context, key string) (Pardon, bool, error) {
	if key == "" {
		return Pardon{}, false, nil
	}

	record, found, err := b.store.Record(ctx, key)
	if err != nil {
		return Pardon{}, false, fmt.Errorf("failed to read the pardon of %q: %w", key, err)
	}
	if !found || !record.Pardon.running(b.clock.Now()) {
		return Pardon{}, false, nil
	}

	return record.Pardon, true, nil
}

func (b *Banner) duration(offence int) time.Duration {
	ladder := b.rules().BanDurations
	if offence > len(ladder) {
		return ladder[len(ladder)-1]
	}
	return ladder[offence-1]
}

func (b *Banner) Flagged(ctx context.Context) (int, error) {
	running, err := b.store.Running(ctx, b.clock.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to count the running bans: %w", err)
	}
	return len(running), nil
}

func (b *Banner) Enforcing() bool { return b.rules().Enforce }
