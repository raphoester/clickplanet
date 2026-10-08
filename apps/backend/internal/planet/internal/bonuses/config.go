package bonuses

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/quizzes"
)

type Config struct {
	MinInterval time.Duration
	MaxInterval time.Duration

	MissRetry time.Duration

	Kinds map[Kind]float64

	OfferTTL time.Duration

	Spread  SpreadConfig
	Bomb    BombConfig
	Enclose EncloseConfig
	Shield  ShieldConfig

	Quiz quizzes.Config

	ActiveWithin time.Duration

	ForgetAfter time.Duration

	MaxChargesPerHour int

	SweepInterval time.Duration
}

type SpreadConfig struct {
	Clicks    int
	MaxPerBox int
}

type BombConfig struct {
	Rings float64
}

type EncloseConfig struct {
	MaxTiles  int
	Held      int
	MaxPerBox int
}

type ShieldConfig struct {
	Held      int
	MinPerBox int
	MaxPerBox int
	PerTile   int
}

const (
	defaultMinInterval       = 4 * time.Minute
	defaultMaxInterval       = 8 * time.Minute
	defaultMissRetry         = 2 * time.Minute
	defaultOfferTTL          = 15 * time.Second
	defaultActiveWithin      = 2 * time.Minute
	defaultForgetAfter       = 5 * time.Minute
	defaultMaxChargesPerHour = 12
	defaultSweepInterval     = time.Second

	defaultSpreadClicks    = 8
	defaultSpreadPerBox    = 4
	defaultEnclosuresHeld  = 3
	defaultEnclosePerBox   = 3
	defaultBombRings       = 4
	defaultEncloseMaxTiles = 25
	defaultShieldsHeld     = 30
	defaultShieldsMinBox   = 5
	defaultShieldsPerBox   = 20
	defaultShieldsOnTile   = 10
)

func (c Config) withDefaults() Config {
	if c.MinInterval <= 0 {
		c.MinInterval = defaultMinInterval
	}
	if c.MaxInterval < c.MinInterval {
		c.MaxInterval = max(c.MinInterval, defaultMaxInterval)
	}
	if len(c.Kinds) == 0 {
		c.Kinds = defaultKinds()
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
	if c.MaxChargesPerHour <= 0 {
		c.MaxChargesPerHour = defaultMaxChargesPerHour
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}

	c.Spread = c.Spread.withDefaults()
	c.Bomb = c.Bomb.withDefaults()
	c.Enclose = c.Enclose.withDefaults()
	c.Shield = c.Shield.withDefaults()

	return c
}

func defaultKinds() map[Kind]float64 {
	return map[Kind]float64{
		KindRefill:        5,
		KindSpreadClicks:  3,
		KindEncloseClicks: 2,
		KindBomb:          1,
		KindShields:       3,
	}
}

func (c SpreadConfig) withDefaults() SpreadConfig {
	if c.Clicks <= 0 {
		c.Clicks = defaultSpreadClicks
	}
	if c.MaxPerBox <= 0 {
		c.MaxPerBox = defaultSpreadPerBox
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
	if c.Held <= 0 {
		c.Held = defaultEnclosuresHeld
	}
	if c.MaxPerBox <= 0 {
		c.MaxPerBox = defaultEnclosePerBox
	}

	return c
}

func (c ShieldConfig) withDefaults() ShieldConfig {
	if c.Held <= 0 {
		c.Held = defaultShieldsHeld
	}
	if c.MinPerBox <= 0 {
		c.MinPerBox = defaultShieldsMinBox
	}
	if c.MaxPerBox < c.MinPerBox {
		c.MaxPerBox = max(c.MinPerBox, defaultShieldsPerBox)
	}
	if c.PerTile <= 0 {
		c.PerTile = defaultShieldsOnTile
	}

	return c
}

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

	return c.Quiz.Validate()
}

func (c Config) ChargesConfig() ChargesConfig {
	c = c.withDefaults()

	return ChargesConfig{
		SpreadClicks:      c.Spread.Clicks,
		Enclosures:        c.Enclose.Held,
		EnclosureMaxTiles: c.Enclose.MaxTiles,
		Shields:           c.Shield.Held,
	}
}

func (c Config) ShieldsPerTile() int {
	return c.Shield.withDefaults().PerTile
}
