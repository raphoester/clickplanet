// Package prune_guests_usecase deletes the guests nobody has used for a long time: one-time visitors and bots.
package prune_guests_usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Pruner interface {
	PruneGuests(ctx context.Context, idleSince time.Time, limit int) (int, error)
}

type Config struct {
	// A guest unused this long is deleted (default 90 days). Never below auth.sessions.guestTTL, or a live cookie loses its account.
	IdleFor time.Duration

	// How often the prune runs (default 1h).
	Interval time.Duration
}

const (
	defaultIdleFor  = 90 * 24 * time.Hour
	defaultInterval = time.Hour
	batch           = 1000
	// Each batch is one statement; a slow database must not hold the loop.
	batchTimeout = 30 * time.Second
)

func (c Config) WithDefaults() Config {
	if c.IdleFor <= 0 {
		c.IdleFor = defaultIdleFor
	}
	if c.Interval <= 0 {
		c.Interval = defaultInterval
	}
	return c
}

type UseCase struct {
	config Config
	guests Pruner
	clock  cptime.Clock
	logger *slog.Logger
}

func New(config Config, guests Pruner, clock cptime.Clock, logger *slog.Logger) *UseCase {
	return &UseCase{config: config.WithDefaults(), guests: guests, clock: clock, logger: logger}
}

func (u *UseCase) Name() string { return "auth-guest-prune" }

// Run prunes once at start, then every interval, and logs what a failed prune left.
func (u *UseCase) Run(ctx context.Context) {
	ticker := time.NewTicker(u.config.Interval)
	defer ticker.Stop()

	for {
		pruned, err := u.Execute(ctx)
		if err != nil && ctx.Err() == nil {
			u.logger.Error("failed to prune the idle guests", slog.Int("pruned", pruned), slog.Any("error", err))
		} else if pruned > 0 {
			u.logger.Info("pruned idle guests", slog.Int("pruned", pruned))
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

// Execute deletes every guest idle past IdleFor, a batch at a time, and says how many.
func (u *UseCase) Execute(ctx context.Context) (int, error) {
	idleSince := u.clock.Now().Add(-u.config.IdleFor)

	total := 0
	for {
		pruned, err := u.pruneBatch(ctx, idleSince)
		total += pruned
		if err != nil {
			return total, err
		}
		if pruned < batch {
			return total, nil
		}
	}
}

func (u *UseCase) pruneBatch(ctx context.Context, idleSince time.Time) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, batchTimeout)
	defer cancel()

	pruned, err := u.guests.PruneGuests(ctx, idleSince, batch)
	if err != nil {
		return 0, fmt.Errorf("failed to prune a batch: %w", err)
	}
	return pruned, nil
}
