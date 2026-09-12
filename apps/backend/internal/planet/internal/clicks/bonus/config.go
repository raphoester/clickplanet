package bonus

import "time"

// Config is per caller: a global ticker made the rate 1/(interval × players).
type Config struct {
	Enabled bool

	MinInterval time.Duration
	MaxInterval time.Duration

	// One miss only; a second in a row waits the ordinary window.
	MissRetry time.Duration

	OfferTTL   time.Duration
	Duration   time.Duration
	Multiplier float64

	ActiveWithin time.Duration

	ForgetAfter time.Duration

	MaxBoostPerHour time.Duration
	SweepInterval   time.Duration
}

const (
	defaultMinInterval     = 90 * time.Second
	defaultMaxInterval     = 210 * time.Second
	defaultMissRetry       = 45 * time.Second
	defaultOfferTTL        = 15 * time.Second
	defaultDuration        = 60 * time.Second
	defaultMultiplier      = 3
	defaultActiveWithin    = 2 * time.Minute
	defaultForgetAfter     = 5 * time.Minute
	defaultMaxBoostPerHour = 15 * time.Minute
	defaultSweepInterval   = time.Second
)

func (c Config) withDefaults() Config {
	if c.MinInterval <= 0 {
		c.MinInterval = defaultMinInterval
	}
	if c.MaxInterval < c.MinInterval {
		c.MaxInterval = max(c.MinInterval, defaultMaxInterval)
	}
	if c.MissRetry <= 0 {
		c.MissRetry = defaultMissRetry
	}
	if c.OfferTTL <= 0 {
		c.OfferTTL = defaultOfferTTL
	}
	if c.Duration <= 0 {
		c.Duration = defaultDuration
	}
	if c.Multiplier <= 1 {
		c.Multiplier = defaultMultiplier
	}
	if c.ActiveWithin <= 0 {
		c.ActiveWithin = defaultActiveWithin
	}
	if c.ForgetAfter <= 0 {
		c.ForgetAfter = defaultForgetAfter
	}
	if c.MaxBoostPerHour <= 0 {
		c.MaxBoostPerHour = defaultMaxBoostPerHour
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}

	return c
}
