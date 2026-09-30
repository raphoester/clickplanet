// Package inmemory_event_buffer holds the events since the last flush, so a click never waits on postgres.
package inmemory_event_buffer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
)

type Persistence interface {
	Save(ctx context.Context, events []activity.Event) error
}

func New(maxPending int, persistence Persistence) *Buffer {
	return &Buffer{maxPending: maxPending, persistence: persistence}
}

type Buffer struct {
	maxPending  int
	persistence Persistence

	// flushMu keeps the tick and the shutdown flush from saving one batch twice.
	flushMu sync.Mutex

	mu      sync.Mutex
	pending []activity.Event
	dropped int
}

var _ activity.Recorder = (*Buffer)(nil)

// Record drops the event past maxPending, so a postgres that stays away costs bounded memory.
func (b *Buffer) Record(event activity.Event) {
	event = event.Trimmed()

	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.pending) >= b.maxPending {
		b.dropped++
		return
	}

	b.pending = append(b.pending, event)
}

// Flushed is what one flush wrote, and what a full buffer dropped since the last one.
type Flushed struct {
	Written int
	Dropped int
}

func (b *Buffer) Flush(ctx context.Context) (Flushed, error) {
	b.flushMu.Lock()
	defer b.flushMu.Unlock()

	b.mu.Lock()
	batch := b.pending
	b.pending = nil
	flushed := Flushed{Dropped: b.dropped}
	b.dropped = 0
	b.mu.Unlock()

	if len(batch) == 0 {
		return flushed, nil
	}

	if err := b.persistence.Save(ctx, batch); err != nil {
		b.keep(batch)
		return flushed, fmt.Errorf("failed to save %d events: %w", len(batch), err)
	}

	flushed.Written = len(batch)

	return flushed, nil
}

// keep puts a failed batch back ahead of what came meanwhile; past maxPending the newest go, as in Record.
func (b *Buffer) keep(batch []activity.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	batch = append(batch, b.pending...)
	if over := len(batch) - b.maxPending; over > 0 {
		batch = batch[:b.maxPending]
		b.dropped += over
	}

	b.pending = batch
}

// Flusher is the flush, as the runner calls it and a decorator wraps it.
type Flusher interface {
	Flush(ctx context.Context) (Flushed, error)
}

const flushTimeout = 10 * time.Second

func NewRunner(interval time.Duration, flusher Flusher) *Runner {
	return &Runner{interval: interval, flusher: flusher}
}

// Runner flushes every interval, and once more when the process stops.
type Runner struct {
	interval time.Duration
	flusher  Flusher
}

func (r *Runner) Name() string { return "activity-buffer" }

func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.flush(ctx)
		case <-ctx.Done():
			r.flush(context.WithoutCancel(ctx))
			return
		}
	}
}

func (r *Runner) flush(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, flushTimeout)
	defer cancel()

	_, _ = r.flusher.Flush(ctx)
}
