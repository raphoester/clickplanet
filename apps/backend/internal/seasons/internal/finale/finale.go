package finale

import (
	"fmt"
	"math"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

type Config struct {
	RefillMultiplier float64
	BoxInterval      time.Duration

	CheckEvery time.Duration
}

const (
	defaultRefillMultiplier = 3
	defaultBoxInterval      = 2 * time.Minute
	defaultCheckEvery       = time.Second
)

func (c Config) WithDefaults() Config {
	if c.RefillMultiplier == 0 {
		c.RefillMultiplier = defaultRefillMultiplier
	}
	if c.BoxInterval == 0 {
		c.BoxInterval = defaultBoxInterval
	}
	if c.CheckEvery <= 0 {
		c.CheckEvery = defaultCheckEvery
	}
	return c
}

func (c Config) Validate() error {
	if m := c.RefillMultiplier; math.IsNaN(m) || math.IsInf(m, 0) || (m != 0 && m < 1) {
		return fmt.Errorf("finale.refillMultiplier is %v: it must be 1 or more, or unset for %d", m, defaultRefillMultiplier)
	}
	if c.BoxInterval < 0 {
		return fmt.Errorf("finale.boxInterval is %s: it must be above 0, or unset for %s", c.BoxInterval, defaultBoxInterval)
	}
	return nil
}

type Rules struct {
	refillMultiplier float64
	boxInterval      time.Duration
}

func NewRules(config Config) Rules {
	config = config.WithDefaults()
	return Rules{refillMultiplier: config.RefillMultiplier, boxInterval: config.BoxInterval}
}

type Switches struct {
	refillMultiplier float64
	boxInterval      time.Duration
	giftTag          string
	giftMadeBefore   time.Time
	frozen           bool
}

func (s Switches) RefillMultiplier() float64 { return s.refillMultiplier }

func (s Switches) BoxInterval() time.Duration { return s.boxInterval }

func (s Switches) Gift() (tag string, madeBefore time.Time, giving bool) {
	return s.giftTag, s.giftMadeBefore, s.giftTag != ""
}

func (s Switches) Frozen() bool { return s.frozen }

func (s Switches) Equal(other Switches) bool {
	return s.refillMultiplier == other.refillMultiplier &&
		s.boxInterval == other.boxInterval &&
		s.giftTag == other.giftTag &&
		s.giftMadeBefore.Equal(other.giftMadeBefore) &&
		s.frozen == other.frozen
}

type stage int

const (
	playing stage = iota
	running
	over
)

type Phase struct {
	stage  stage
	season calendar.Season
}

func PhaseAt(seasons calendar.Calendar, now time.Time) Phase {
	if season, ok := seasons.Current(now); ok {
		if now.Before(season.FinaleStartsAt) {
			return Phase{stage: playing, season: season}
		}
		return Phase{stage: running, season: season}
	}

	if season, ok := seasons.LastEnded(now); ok {
		return Phase{stage: over, season: season}
	}

	return Phase{stage: playing}
}

func (p Phase) Running() bool { return p.stage == running }

func (p Phase) Season() calendar.Season { return p.season }

// Each account is given once per finale: the tag names the finale, and planet keeps who has had it.
func (p Phase) Switches(rules Rules) Switches {
	switch p.stage {
	case running:
		return Switches{
			refillMultiplier: rules.refillMultiplier,
			boxInterval:      rules.boxInterval,
			giftTag:          fmt.Sprintf("season-%d-finale", p.season.Number),
			giftMadeBefore:   p.season.FinaleStartsAt,
		}
	case over:
		return Switches{frozen: true}
	default:
		return Switches{}
	}
}
