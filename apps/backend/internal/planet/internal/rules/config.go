package rules

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"
)

type Config struct {
	Sets map[SetName]SetConfig
}

type SetConfig struct {
	Tempo float64

	Boxes BoxesConfig

	AntiBot AntiBotConfig

	Frozen bool
}

type AntiBotConfig struct {
	Crowd int
}

type BoxesConfig struct {
	MinInterval time.Duration
	MaxInterval time.Duration

	MaxChargesPerHour int
}

var setNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

func (c Config) Validate() error {
	var errs []error
	for name, set := range c.Sets {
		if name == Default {
			errs = append(errs, fmt.Errorf("rules.sets.%s: %q is the config's own rules and cannot be redefined", name, Default))
		}
		if !setNamePattern.MatchString(string(name)) {
			errs = append(errs, fmt.Errorf("rules.sets.%s: a name is a lowercase letter, then up to 31 of a-z, 0-9, - and _", name))
		}
		if err := set.validate(); err != nil {
			errs = append(errs, fmt.Errorf("rules.sets.%s: %w", name, err))
		}
	}

	return errors.Join(errs...)
}

func (c SetConfig) validate() error {
	if math.IsNaN(c.Tempo) || math.IsInf(c.Tempo, 0) || c.Tempo < 0 {
		return fmt.Errorf("tempo is %v: it must be above 0, or unset for the config's", c.Tempo)
	}
	if c.Boxes.MinInterval < 0 || c.Boxes.MaxInterval < 0 || c.Boxes.MaxChargesPerHour < 0 {
		return errors.New("boxes must not be negative")
	}
	if c.AntiBot.Crowd < 0 {
		return fmt.Errorf("antiBot.crowd is %d: it must be above 0, or unset for the config's", c.AntiBot.Crowd)
	}

	return nil
}
