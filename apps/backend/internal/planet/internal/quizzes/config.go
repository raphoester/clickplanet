package quizzes

import (
	"fmt"
	"math"
	"time"
)

// Config is the quiz's own numbers, per caller. It sits under `bonus.quiz` because a right answer
// pays out in the same charges a box does, but the schedule is its own: a quiz is a second way to
// earn one, not a box of a different shape.
type Config struct {
	// Off by default. A deploy that has not run `make quiz` still boots; it just never asks.
	Enabled bool

	MinInterval time.Duration
	MaxInterval time.Duration

	// How long the banner sits there unopened before it is gone. Generous, because it is only an
	// invitation: the clock that matters starts at OpenQuiz.
	OfferTTL time.Duration

	// How long the player has once the question is on screen. Short on purpose — the whole point is
	// to be answered from what somebody knows rather than from what they can look up.
	AnswerWindow time.Duration

	// How much harder the draw leans on the countries that are winning. **Zero is a flat draw** and
	// is also the zero value, so a config that does not mention it asks about every country the
	// bank covers equally — the safe reading of a number left out. 1 makes a country holding an
	// even slice of the map about twice as likely to be asked about as one holding nothing.
	// See weights.go.
	LeaderBias float64

	// The most charges one caller may win from quizzes per hour. Its own budget, separate from the
	// boxes': a player who answers everything gets this many on top, and no more.
	MaxChargesPerHour int
}

const (
	// A little rarer than the boxes, so the two together are roughly one offer every two minutes
	// of active play rather than a thing that never stops interrupting.
	defaultMinInterval = 6 * time.Minute
	defaultMaxInterval = 11 * time.Minute

	// Long enough to finish the click you were making and look up.
	defaultOfferTTL = 25 * time.Second

	// Long enough to read three choices, and short enough that looking the answer up is not worth
	// it. That second half is the guideline the number answers to; retune it freely within it.
	defaultAnswerWindow = 8 * time.Second

	defaultMaxChargesPerHour = 6
)

func (c Config) withDefaults() Config {
	if c.MinInterval <= 0 {
		c.MinInterval = defaultMinInterval
	}
	if c.MaxInterval < c.MinInterval {
		c.MaxInterval = max(c.MinInterval, defaultMaxInterval)
	}
	if c.OfferTTL <= 0 {
		c.OfferTTL = defaultOfferTTL
	}
	if c.AnswerWindow <= 0 {
		c.AnswerWindow = defaultAnswerWindow
	}
	if c.MaxChargesPerHour <= 0 {
		c.MaxChargesPerHour = defaultMaxChargesPerHour
	}

	return c
}

// Validate refuses numbers that would make a quiz unanswerable or a lean that is not a number,
// rather than finding out at the first offer.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.LeaderBias < 0 || math.IsNaN(c.LeaderBias) || math.IsInf(c.LeaderBias, 0) {
		return fmt.Errorf("bonus.quiz.leaderBias is %v: a lean must be a number of 0 or more", c.LeaderBias)
	}

	if c.AnswerWindow < 0 {
		return fmt.Errorf("bonus.quiz.answerWindow is %v: a player cannot answer in less than no time",
			c.AnswerWindow)
	}

	if c.MinInterval < 0 || c.OfferTTL < 0 {
		return fmt.Errorf("bonus.quiz.minInterval and offerTtl must not be negative")
	}

	return nil
}

// Settings is the config with its defaults filled in, for whoever schedules the offers.
func (c Config) Settings() Config { return c.withDefaults() }
