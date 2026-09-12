package bonus

import "time"

type Config struct {
	Enabled bool

	// How often a box is put in front of somebody. Nothing is offered while
	// nobody is watching, so this is a ceiling rather than a schedule.
	Interval time.Duration

	// How long the token stays good. It has to outlast the flight the client
	// draws, or a box caught on the last frame is refused.
	OfferTTL time.Duration

	// How long a caught bonus runs for.
	Duration time.Duration

	// What it multiplies the click allowance by.
	Multiplier float64
}

const (
	defaultInterval   = 90 * time.Second
	defaultOfferTTL   = 15 * time.Second
	defaultDuration   = 60 * time.Second
	defaultMultiplier = 3
)

func (c Config) withDefaults() Config {
	if c.Interval <= 0 {
		c.Interval = defaultInterval
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
	return c
}
