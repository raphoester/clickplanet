package rules

import (
	"errors"
	"fmt"
	"math"
	"regexp"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

type SetName string

// Default is the set the migration keeps the game's own rules in.
const Default SetName = "default"

var (
	ErrUnknownSet = errors.New("no rule set has this name")
	ErrInvalidSet = errors.New("this is not a rule set")
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

// minCrowd is the cohort watchdog's smallest group: a crowd of one network is no crowd.
const minCrowd = 2

type Set struct {
	name  SetName
	tempo float64
	boxes bonuses.Pace
	crowd int
}

func (s Set) TakeSnapshot() Snapshot {
	return Snapshot{Name: s.name, Tempo: s.tempo, Boxes: s.boxes, Crowd: s.crowd}
}

type Snapshot struct {
	Name SetName

	// How many times faster than rateLimiter.perSecond every bucket refills.
	Tempo float64

	Boxes bonuses.Pace

	// How many networks of one prefix, in step, are banned as a crowd.
	Crowd int
}

func (s Snapshot) Restore() (Set, error) {
	switch {
	case !namePattern.MatchString(string(s.Name)):
		return Set{}, fmt.Errorf("%w: a name is a lowercase letter, then up to 31 of a-z, 0-9, - and _", ErrInvalidSet)
	case math.IsNaN(s.Tempo) || math.IsInf(s.Tempo, 0) || s.Tempo <= 0:
		return Set{}, fmt.Errorf("%w: its tempo is %v, and must be above 0", ErrInvalidSet, s.Tempo)
	case s.Boxes.MinInterval <= 0 || s.Boxes.MaxInterval < s.Boxes.MinInterval:
		return Set{}, fmt.Errorf("%w: a box waits %s at least and %s at most", ErrInvalidSet,
			s.Boxes.MinInterval, s.Boxes.MaxInterval)
	case s.Boxes.MaxChargesPerHour <= 0:
		return Set{}, fmt.Errorf("%w: its hourly cap is %d, and must be above 0", ErrInvalidSet, s.Boxes.MaxChargesPerHour)
	case s.Crowd < minCrowd:
		return Set{}, fmt.Errorf("%w: a crowd is %d networks, and must be %d or more", ErrInvalidSet, s.Crowd, minCrowd)
	}

	return Set{name: s.Name, tempo: s.Tempo, boxes: s.Boxes, crowd: s.Crowd}, nil
}
