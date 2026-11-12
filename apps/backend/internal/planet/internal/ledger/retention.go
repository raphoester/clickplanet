package ledger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrShortRetention = errors.New("the ledger keeps less than the recent history, which reads postgres from its head")

type Config struct {
	Retention     time.Duration
	SweepInterval time.Duration
}

const (
	defaultRetention     = 72 * time.Hour
	defaultSweepInterval = 5 * time.Minute
)

func (c Config) Validate() error {
	if c.Retention > 0 && c.Retention < recentSpan+recentStep {
		return fmt.Errorf("%w: %s", ErrShortRetention, c.Retention)
	}
	return nil
}

func (c Config) withDefaults() Config {
	if c.Retention <= 0 {
		c.Retention = defaultRetention
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

func NewRetention(config Config, takings Storage, clock cptime.Clock) *Retention {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Retention{config: config.withDefaults(), takings: takings, clock: clock}
}

type Retention struct {
	config  Config
	takings Storage
	clock   cptime.Clock
}

func (r *Retention) Name() string { return "tile-ledger" }

func (r *Retention) Run(ctx context.Context) {
	ticker := time.NewTicker(r.config.SweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.Sweep()
		case <-ctx.Done():
			return
		}
	}
}

func (r *Retention) Sweep() {
	r.takings.ForgetBefore(r.clock.Now().Add(-r.config.Retention))
}
