package bonus

import (
	"fmt"
	"math"
	"slices"
	"time"
)

// Config is per caller: a global ticker made the rate 1/(interval × players).
type Config struct {
	Enabled bool

	MinInterval time.Duration
	MaxInterval time.Duration

	// One miss only; a second in a row waits the ordinary window.
	MissRetry time.Duration

	// What a box can be worth. Each kind is drawn with a chance of its weight over
	// the sum of the weights, so {triple_clicks: 3, spread_clicks: 1} makes one
	// box in four a spread. A kind left out, or at 0, is never offered. Empty
	// offers every kind equally.
	Kinds map[Kind]float64

	OfferTTL time.Duration

	// How long a caught bonus runs. SpreadDuration is spread_clicks' own, much
	// shorter: a click that takes seven tiles is worth far more than three clicks.
	Duration       time.Duration
	SpreadDuration time.Duration

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
	defaultDuration        = 20 * time.Second
	defaultSpreadDuration  = 10 * time.Second
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
	if len(c.Kinds) == 0 {
		c.Kinds = make(map[Kind]float64, len(Kinds))
		for _, kind := range Kinds {
			c.Kinds[kind] = 1
		}
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
	if c.SpreadDuration <= 0 {
		c.SpreadDuration = defaultSpreadDuration
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

// Validate refuses a kind this server cannot grant, and weights that could never
// draw anything, rather than offering boxes nobody asked for.
func (c Config) Validate() error {
	total := 0.0

	for kind, weight := range c.Kinds {
		if !slices.Contains(Kinds, kind) {
			return fmt.Errorf("bonus.kinds holds %q, which is not one of %v", kind, Kinds)
		}
		if weight < 0 || math.IsNaN(weight) || math.IsInf(weight, 0) {
			return fmt.Errorf("bonus.kinds.%s is %v: a weight must be a number of 0 or more", kind, weight)
		}
		total += weight
	}

	if len(c.Kinds) > 0 && total == 0 {
		return fmt.Errorf("bonus.kinds gives every kind a weight of 0, so no box could be anything")
	}

	return nil
}

func (c Config) durationOf(kind Kind) time.Duration {
	if kind == KindSpreadClicks {
		return c.SpreadDuration
	}

	return c.Duration
}
