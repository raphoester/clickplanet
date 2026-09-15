package bonuses

import (
	"fmt"
	"math"
	"slices"
	"time"
)

// Config is per caller: a global ticker made the rate 1/(interval × players).
type Config struct {
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

	Triple  TripleConfig
	Spread  SpreadConfig
	Bomb    BombConfig
	Enclose EncloseConfig

	ActiveWithin time.Duration

	ForgetAfter time.Duration

	MaxBoostPerHour time.Duration
	SweepInterval   time.Duration
}

type TripleConfig struct {
	Duration   time.Duration
	Multiplier float64
}

// Much shorter than a triple: a click that takes seven tiles is worth far more than three clicks.
type SpreadConfig struct {
	Duration time.Duration
}

type BombConfig struct {
	// How long a bomb may be held before it is lost; it counts towards MaxBoostPerHour like any bonus.
	Duration time.Duration

	// How wide a circle a bomb clears, in tile spacings: 10.4 is ~390 tiles inland.
	Rings float64
}

// A shape bigger than MaxTiles takes nothing, and costs nothing.
type EncloseConfig struct {
	Duration time.Duration
	Shapes   int
	MaxTiles int
}

const (
	defaultMinInterval     = 90 * time.Second
	defaultMaxInterval     = 210 * time.Second
	defaultMissRetry       = 45 * time.Second
	defaultOfferTTL        = 15 * time.Second
	defaultActiveWithin    = 2 * time.Minute
	defaultForgetAfter     = 5 * time.Minute
	defaultMaxBoostPerHour = 15 * time.Minute
	defaultSweepInterval   = time.Second

	defaultTripleDuration  = 20 * time.Second
	defaultMultiplier      = 3
	defaultSpreadDuration  = 10 * time.Second
	defaultBombDuration    = 30 * time.Second
	defaultBombRings       = 10.4
	defaultEncloseDuration = 30 * time.Second
	defaultEncloseShapes   = 3
	defaultEncloseMaxTiles = 15
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

	c.Triple = c.Triple.withDefaults()
	c.Spread = c.Spread.withDefaults()
	c.Bomb = c.Bomb.withDefaults()
	c.Enclose = c.Enclose.withDefaults()

	return c
}

func (c TripleConfig) withDefaults() TripleConfig {
	if c.Duration <= 0 {
		c.Duration = defaultTripleDuration
	}
	if c.Multiplier <= 1 {
		c.Multiplier = defaultMultiplier
	}

	return c
}

func (c SpreadConfig) withDefaults() SpreadConfig {
	if c.Duration <= 0 {
		c.Duration = defaultSpreadDuration
	}

	return c
}

func (c BombConfig) withDefaults() BombConfig {
	if c.Duration <= 0 {
		c.Duration = defaultBombDuration
	}
	if c.Rings <= 0 {
		c.Rings = defaultBombRings
	}

	return c
}

func (c EncloseConfig) withDefaults() EncloseConfig {
	if c.Duration <= 0 {
		c.Duration = defaultEncloseDuration
	}
	if c.Shapes <= 0 {
		c.Shapes = defaultEncloseShapes
	}
	if c.MaxTiles <= 0 {
		c.MaxTiles = defaultEncloseMaxTiles
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
	switch kind {
	case KindSpreadClicks:
		return c.Spread.Duration
	case KindBomb:
		return c.Bomb.Duration
	case KindEncloseClicks:
		return c.Enclose.Duration
	case KindTripleClicks:
	}

	return c.Triple.Duration
}
