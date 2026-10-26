package rules

import (
	"errors"
	"maps"
	"reflect"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type SetName string

// Default is the set the migration keeps the game's own rules in.
const Default SetName = "default"

var (
	ErrUnknownSet = errors.New("no rule set has this name")
	ErrInvalidSet = errors.New("this is not a rule set")
)

type Set struct {
	name    SetName
	tempo   float64
	clicks  clicks.Throttle
	toll    []clicks.TollStep
	bonuses bonuses.Rules
}

func (s Set) TakeSnapshot() Snapshot {
	return Snapshot{
		Name:    s.name,
		Tempo:   s.tempo,
		Clicks:  s.clicks,
		Toll:    slices.Clone(s.toll),
		Bonuses: cloned(s.bonuses),
	}
}

type Snapshot struct {
	Name SetName

	// How many times faster than Clicks.PerSecond every bucket refills.
	Tempo float64

	Clicks  clicks.Throttle
	Toll    []clicks.TollStep
	Bonuses bonuses.Rules
}

func (s Snapshot) Restore() (Set, error) {
	if err := s.problem(); err != nil {
		return Set{}, err
	}

	return Set{
		name:    s.Name,
		tempo:   s.Tempo,
		clicks:  s.Clicks,
		toll:    slices.Clone(s.Toll),
		bonuses: cloned(s.Bonuses),
	}, nil
}

func (s Snapshot) Equal(other Snapshot) bool {
	return reflect.DeepEqual(s, other)
}

func cloned(rules bonuses.Rules) bonuses.Rules {
	rules.Kinds = maps.Clone(rules.Kinds)
	return rules
}
