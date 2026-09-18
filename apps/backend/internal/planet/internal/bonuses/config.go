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

	// The most triple clicks time one caller may be granted per hour.
	MaxBoostPerHour time.Duration

	// The most charges (bomb, enclose, spread) one caller may be granted per hour. A charge has no time to
	// count, so it is counted apart: a script that catches every box gets this many, and no more.
	MaxChargesPerHour int

	// How long a charge is kept unspent before it is lost. Long, so a charge is a reason to come back.
	ChargeTTL time.Duration

	SweepInterval time.Duration
}

type TripleConfig struct {
	Duration   time.Duration
	Multiplier float64
}

// A spread charge is a number of clicks, not a time: 8 clicks of 7 tiles is about a bomb's worth, and a
// timer only rewarded whoever could dump a full bank of clicks inside it.
type SpreadConfig struct {
	Clicks int
}

type BombConfig struct {
	// How wide a circle a bomb clears, in tile spacings: 10.4 is ~390 tiles inland.
	Rings float64
}

// An enclose charge is one shape. A shape bigger than MaxTiles takes nothing, and costs nothing.
type EncloseConfig struct {
	MaxTiles int
}

const (
	defaultMinInterval       = 90 * time.Second
	defaultMaxInterval       = 210 * time.Second
	defaultMissRetry         = 45 * time.Second
	defaultOfferTTL          = 15 * time.Second
	defaultActiveWithin      = 2 * time.Minute
	defaultForgetAfter       = 5 * time.Minute
	defaultMaxBoostPerHour   = 15 * time.Minute
	defaultMaxChargesPerHour = 6
	defaultChargeTTL         = 24 * time.Hour
	defaultSweepInterval     = time.Second

	defaultTripleDuration  = 20 * time.Second
	defaultMultiplier      = 3
	defaultSpreadClicks    = 8
	defaultBombRings       = 10.4
	defaultEncloseMaxTiles = 25
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
	if c.MaxChargesPerHour <= 0 {
		c.MaxChargesPerHour = defaultMaxChargesPerHour
	}
	if c.ChargeTTL <= 0 {
		c.ChargeTTL = defaultChargeTTL
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
	if c.Clicks <= 0 {
		c.Clicks = defaultSpreadClicks
	}

	return c
}

func (c BombConfig) withDefaults() BombConfig {
	if c.Rings <= 0 {
		c.Rings = defaultBombRings
	}

	return c
}

func (c EncloseConfig) withDefaults() EncloseConfig {
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

// durationOf is how long a kind runs: zero for a charge, which runs until it is spent.
func (c Config) durationOf(kind Kind) time.Duration {
	if !kind.Timed() {
		return 0
	}

	return c.Triple.Duration
}

// charges is the part of the config the charges read, defaults filled in.
func (c Config) charges() ChargesConfig {
	return ChargesConfig{TTL: c.ChargeTTL, SpreadClicks: c.Spread.Clicks, EnclosureMaxTiles: c.Enclose.MaxTiles}
}
