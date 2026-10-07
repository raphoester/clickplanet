package quizzes

import (
	"fmt"
	"math"
	"time"
)

type Config struct {
	Enabled bool

	MinInterval time.Duration
	MaxInterval time.Duration

	MissRetry time.Duration

	OfferTTL time.Duration

	AnswerWindow time.Duration

	LeaderBias float64

	MaxChargesPerHour int
}

const (
	defaultMinInterval = 4 * time.Minute
	defaultMaxInterval = 8 * time.Minute

	defaultMissRetry = 2 * time.Minute

	defaultOfferTTL = 25 * time.Second

	defaultAnswerWindow = 8 * time.Second

	defaultMaxChargesPerHour = 12
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
	if c.AnswerWindow <= 0 {
		c.AnswerWindow = defaultAnswerWindow
	}
	if c.MaxChargesPerHour <= 0 {
		c.MaxChargesPerHour = defaultMaxChargesPerHour
	}

	return c
}

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

	if c.MinInterval < 0 || c.MissRetry < 0 || c.OfferTTL < 0 {
		return fmt.Errorf("bonus.quiz.minInterval, missRetry and offerTtl must not be negative")
	}

	return nil
}

func (c Config) Settings() Config { return c.withDefaults() }
