package inmemory_player_storage

import (
	"context"
	"time"
)

// Flusher is the flush, as the runner calls it and a decorator wraps it.
type Flusher interface {
	Flush(ctx context.Context) error
}

// flushTimeout bounds one flush, so a stuck connection cannot stall the loop.
const flushTimeout = 10 * time.Second

// Runner flushes every interval, and once more when it stops. A failed flush is retried on the next tick;
// what a flush did is its decorators' to report.
type Runner struct {
	interval time.Duration
	flusher  Flusher
}

func NewRunner(config Config, flusher Flusher) *Runner {
	return &Runner{interval: config.WithDefaults().FlushInterval, flusher: flusher}
}

func (r *Runner) Name() string { return "player-storage" }

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

	_ = r.flusher.Flush(ctx)
}
