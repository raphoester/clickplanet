package activity

import (
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

type Config struct {
	Enabled  bool
	Database cppg.Config

	FlushInterval time.Duration
	MaxPending    int

	// The privacy policy promises 72 hours: change it before this.
	Retention     time.Duration
	SweepInterval time.Duration
	MaxEvents     int
}

const (
	defaultFlushInterval = time.Second
	defaultMaxPending    = 100_000
	defaultRetention     = 72 * time.Hour
	defaultSweepInterval = 5 * time.Minute
	defaultMaxEvents     = 10_000_000
)

func (c Config) WithDefaults() Config {
	if c.FlushInterval <= 0 {
		c.FlushInterval = defaultFlushInterval
	}
	if c.MaxPending <= 0 {
		c.MaxPending = defaultMaxPending
	}
	if c.Retention <= 0 {
		c.Retention = defaultRetention
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	if c.MaxEvents <= 0 {
		c.MaxEvents = defaultMaxEvents
	}

	return c
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("activity.database: %w", err)
	}

	if c.FlushInterval < 0 || c.MaxPending < 0 || c.Retention < 0 || c.SweepInterval < 0 || c.MaxEvents < 0 {
		return errors.New("activity: flushInterval, maxPending, retention, sweepInterval and maxEvents may not be negative")
	}

	return nil
}
