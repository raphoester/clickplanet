package rules

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/catcher"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/churner"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/cohort"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/defender"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/hopper"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/jury"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/metronome"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/retaker"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rotator"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/scraper"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/sequencer"
)

type SetName string

// Default is the set the migration keeps production's rules in, and the one played when no set has the name asked.
const Default SetName = "default"

var (
	ErrUnknownSet = errors.New("no rule set has this name")
	ErrInvalidSet = errors.New("this rule set cannot play")
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

type Set struct {
	name SetName

	bans      bans.Rules
	jury      jury.Rules
	retaker   retaker.Rules
	sequencer sequencer.Rules
	metronome metronome.Rules
	defender  defender.Rules
	catcher   catcher.Rules
	cohort    cohort.Rules
	scraper   scraper.Rules
	churner   churner.Rules
	hopper    hopper.Rules
	rotator   rotator.Rules
}

func (s Set) TakeSnapshot() Snapshot {
	return Snapshot{
		Name:      s.name,
		Bans:      s.bans,
		Jury:      s.jury,
		Retaker:   s.retaker,
		Sequencer: s.sequencer,
		Metronome: s.metronome,
		Defender:  s.defender,
		Catcher:   s.catcher,
		Cohort:    s.cohort,
		Scraper:   s.scraper,
		Churner:   s.churner,
		Hopper:    s.hopper,
		Rotator:   s.rotator,
	}
}

// Equal is true when both sets have one name and judge alike.
func (s Set) Equal(other Set) bool {
	return reflect.DeepEqual(s, other)
}

type Snapshot struct {
	Name SetName

	Bans      bans.Rules
	Jury      jury.Rules
	Retaker   retaker.Rules
	Sequencer sequencer.Rules
	Metronome metronome.Rules
	Defender  defender.Rules
	Catcher   catcher.Rules
	Cohort    cohort.Rules
	Scraper   scraper.Rules
	Churner   churner.Rules
	Hopper    hopper.Rules
	Rotator   rotator.Rules
}

// Restore refuses a set with any value its watchdog cannot judge with: nothing falls back to a default.
func (s Snapshot) Restore() (Set, error) {
	if !namePattern.MatchString(string(s.Name)) {
		return Set{}, fmt.Errorf("%w: a name is a lowercase letter, then up to 31 of a-z, 0-9, - and _", ErrInvalidSet)
	}

	var reasons []string
	for _, part := range []struct {
		name string
		err  error
	}{
		{"bans", s.Bans.Validate()},
		{"jury", s.Jury.Validate()},
		{"retaker", s.Retaker.Validate()},
		{"sequencer", s.Sequencer.Validate()},
		{"metronome", s.Metronome.Validate()},
		{"defender", s.Defender.Validate()},
		{"catcher", s.Catcher.Validate()},
		{"cohort", s.Cohort.Validate()},
		{"scraper", s.Scraper.Validate()},
		{"churner", s.Churner.Validate()},
		{"hopper", s.Hopper.Validate()},
		{"rotator", s.Rotator.Validate()},
	} {
		reasons = append(reasons, reasonsOf(part.name, part.err)...)
	}
	if len(reasons) > 0 {
		return Set{}, fmt.Errorf("%w: %s", ErrInvalidSet, strings.Join(reasons, "; "))
	}

	return Set{
		name:      s.Name,
		bans:      s.Bans,
		jury:      s.Jury,
		retaker:   s.Retaker,
		sequencer: s.Sequencer,
		metronome: s.Metronome,
		defender:  s.Defender,
		catcher:   s.Catcher,
		cohort:    s.Cohort,
		scraper:   s.Scraper,
		churner:   s.Churner,
		hopper:    s.Hopper,
		rotator:   s.Rotator,
	}, nil
}

// Each reason names its value by its path in the set, as antibot.v1.Rules writes it.
func reasonsOf(part string, err error) []string {
	if err == nil {
		return nil
	}

	var joined interface{ Unwrap() []error }
	if !errors.As(err, &joined) {
		return []string{part + "." + err.Error()}
	}

	var reasons []string
	for _, each := range joined.Unwrap() {
		reasons = append(reasons, reasonsOf(part, each)...)
	}
	return reasons
}
