package bonuses

import "time"

type Config struct {
	ForgetAfter time.Duration

	SweepInterval time.Duration
}

const (
	defaultForgetAfter   = 5 * time.Minute
	defaultSweepInterval = time.Second
)

func (c Config) withDefaults() Config {
	if c.ForgetAfter <= 0 {
		c.ForgetAfter = defaultForgetAfter
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}

	return c
}
