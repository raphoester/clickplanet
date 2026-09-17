package shadowban

import (
	"context"
	"fmt"
	"time"
)

// Record is one scope's or one account's ban as it is kept between boots. When it may be flagged again is not kept.
type Record struct {
	Key      string
	Flags    int
	Offences int
	Until    time.Time
}

// Persistence is where the bans are kept between boots. It is never read after Load.
type Persistence interface {
	Load(ctx context.Context, visit func(record Record)) error
	Save(ctx context.Context, records []Record) error
}

const flushTimeout = 10 * time.Second

// Load refuses rather than start empty: a boot that forgets the bans unbans every bot.
func (b *Banner) Load(ctx context.Context) error {
	bans := make(map[string]*ban)
	if err := b.persistence.Load(ctx, func(record Record) {
		bans[record.Key] = record.ban()
	}); err != nil {
		return fmt.Errorf("failed to read stored bans: %w", err)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.bans = bans

	return nil
}

func (r Record) ban() *ban {
	return &ban{flags: r.Flags, offences: r.Offences, until: r.Until}
}

func (b *Banner) Run(ctx context.Context) {
	ticker := time.NewTicker(b.config.SaveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.flushOrReport(ctx)
		case <-ctx.Done():
			b.flushOrReport(context.WithoutCancel(ctx))
			return
		}
	}
}

func (b *Banner) flushOrReport(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, flushTimeout)
	defer cancel()

	if err := b.Flush(ctx); err != nil {
		b.onStateError(fmt.Errorf("failed to flush bans, retrying next tick: %w", err))
	}
}

// Flush writes every ban changed since the last flush, as it is now. A failed write keeps them marked.
func (b *Banner) Flush(ctx context.Context) error {
	records := b.takeDirty()
	if len(records) == 0 {
		return nil
	}

	if err := b.persistence.Save(ctx, records); err != nil {
		b.mu.Lock()
		for _, record := range records {
			b.dirty.Add(record.Key)
		}
		b.mu.Unlock()

		return fmt.Errorf("failed to save %d bans: %w", len(records), err)
	}

	return nil
}

func (b *Banner) takeDirty() []Record {
	b.mu.Lock()
	defer b.mu.Unlock()

	records := make([]Record, 0, b.dirty.Len())
	b.dirty.ForEach(func(scope string) {
		record := b.bans[scope]
		records = append(records, Record{
			Key:      scope,
			Flags:    record.flags,
			Offences: record.offences,
			Until:    record.until,
		})
	})
	b.dirty.Clear()

	return records
}
