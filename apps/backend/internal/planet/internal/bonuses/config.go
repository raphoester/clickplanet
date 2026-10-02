package bonuses

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/quizzes"
)

// Config is per caller: a global ticker made the rate 1/(interval × players).
type Config struct {
	MinInterval time.Duration
	MaxInterval time.Duration

	// One miss only; a second in a row waits the ordinary window.
	MissRetry time.Duration

	// What a box can be worth. Each kind is drawn with a chance of its weight over
	// the sum of the weights, so {refill: 3, spread_clicks: 1} makes one
	// box in four a spread. A kind left out, or at 0, is never offered. Empty
	// takes defaultKinds.
	Kinds map[Kind]float64

	OfferTTL time.Duration

	Spread  SpreadConfig
	Bomb    BombConfig
	Enclose EncloseConfig

	// The quizzes: a second way to earn one of the charges above, on a schedule of its own. Off
	// unless switched on. See the quizzes package.
	Quiz quizzes.Config

	ActiveWithin time.Duration

	ForgetAfter time.Duration

	// The most charges one caller may be granted per hour: a script that catches every box gets this many,
	// and no more.
	MaxChargesPerHour int

	SweepInterval time.Duration
}

// A spread is a pool of clicks, not a time: a timer only rewarded whoever could dump a full bank of clicks
// inside it. A box adds 1 to MaxPerBox clicks, drawn at the claim, up to Clicks: 8 clicks of 7 tiles is
// about a bomb's worth.
type SpreadConfig struct {
	Clicks    int
	MaxPerBox int
}

type BombConfig struct {
	// How wide a circle a bomb clears, in tile spacings: 4 is ~57 tiles inland, about one bank.
	Rings float64
}

// An enclose charge is one shape, and a player stacks up to Held of them. A box adds 1 to MaxPerBox, drawn
// at the claim. A shape bigger than MaxTiles takes nothing, and costs nothing.
type EncloseConfig struct {
	MaxTiles  int
	Held      int
	MaxPerBox int
}

// Sized in banks: one full click allowance, 60 clicks at one every 5s. A bomb never
// clears more than one player can take back with one bank.
const (
	// About ten boxes an hour, and a caught one pushes the next a window further.
	defaultMinInterval  = 4 * time.Minute
	defaultMaxInterval  = 8 * time.Minute
	defaultMissRetry    = 2 * time.Minute
	defaultOfferTTL     = 15 * time.Second
	defaultActiveWithin = 2 * time.Minute
	defaultForgetAfter  = 5 * time.Minute
	// Above the ten boxes an hour a person who catches every one gets: only the cap on
	// a script, which does. It only stops the next offer.
	defaultMaxChargesPerHour = 12
	defaultSweepInterval     = time.Second

	defaultSpreadClicks    = 8
	defaultSpreadPerBox    = 4
	defaultEnclosuresHeld  = 3
	defaultEnclosePerBox   = 3
	defaultBombRings       = 4
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

	return c
}

// About one bomb an hour of active play.
func defaultKinds() map[Kind]float64 {
	return map[Kind]float64{
		KindRefill:        5,
		KindSpreadClicks:  3,
		KindEncloseClicks: 2,
		KindBomb:          1,
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

	return c.Quiz.Validate()
}

// ChargesConfig is the part of the config the charges read, defaults filled in.
func (c Config) ChargesConfig() ChargesConfig {
	c = c.withDefaults()

	return ChargesConfig{SpreadClicks: c.Spread.Clicks, Enclosures: c.Enclose.Held, EnclosureMaxTiles: c.Enclose.MaxTiles}
}
