package posts

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// The arm of each experiment a clip is in, by experiment: {"length": "short"}.
type Variants map[string]string

func (v Variants) String() string {
	arms := make([]string, 0, len(v))
	for _, name := range slices.Sorted(maps.Keys(v)) {
		arms = append(arms, name+" "+v[name])
	}
	return strings.Join(arms, ", ")
}

type Arm struct {
	Name string
	// Its share of the clips is its weight over the sum of the experiment's. 0 is an arm no coin draws: what a clip is
	// moved to when the arm it was drawn for could not be applied (the model's run that the rules made), so it is still
	// counted, apart.
	Weight int
}

// Arms of one weight each: an even split.
func EvenArms(names ...string) []Arm {
	arms := make([]Arm, 0, len(names))
	for _, name := range names {
		arms = append(arms, Arm{Name: name, Weight: 1})
	}
	return arms
}

type Experiment struct {
	Name string
	Arms []Arm
	// The arm that is exactly what a clip was before the experiment.
	Control string
	// One coin for the whole run, flipped before the stories are chosen, and every clip of the run in its arm: for an
	// experiment on which stories are told. Otherwise a coin per clip, after: an arm never changes which story is told.
	PerRun bool
}

func (e Experiment) arm(name string) bool {
	return slices.ContainsFunc(e.Arms, func(arm Arm) bool { return arm.Name == name })
}

// One coin, the arms weighted.
func (e Experiment) flip(dice Dice) string {
	total := 0
	for _, arm := range e.Arms {
		total += arm.Weight
	}
	roll := dice.IntN(total)
	for _, arm := range e.Arms {
		if roll < arm.Weight {
			return arm.Name
		}
		roll -= arm.Weight
	}
	return e.Control
}

// The experiments that run, each on every clip, each with a coin of its own: two at once are a factorial design.
type Experiments []Experiment

var (
	ErrInvalidExperiment = errors.New("the experiment is invalid")
	ErrUnknownVariant    = errors.New("no such experiment or arm")
)

func ExperimentsOf(experiments ...Experiment) (Experiments, error) {
	named := cpcolls.NewSet[string]()
	for _, experiment := range experiments {
		if experiment.Name == "" || named.Contains(experiment.Name) {
			return nil, fmt.Errorf("%w: %q is no name, or the name of two", ErrInvalidExperiment, experiment.Name)
		}
		named.Add(experiment.Name)

		arms, drawn := cpcolls.NewSet[string](), 0
		for _, arm := range experiment.Arms {
			if arm.Name == "" || arms.Contains(arm.Name) || arm.Weight < 0 {
				return nil, fmt.Errorf("%w: %q has an arm %q with no name, twice, or with a weight under 0",
					ErrInvalidExperiment, experiment.Name, arm.Name)
			}
			arms.Add(arm.Name)
			if arm.Weight > 0 {
				drawn++
			}
		}
		if drawn < 2 {
			return nil, fmt.Errorf("%w: %q has fewer than two arms a coin draws", ErrInvalidExperiment, experiment.Name)
		}
		if !arms.Contains(experiment.Control) {
			return nil, fmt.Errorf("%w: %q has no arm %q for its control", ErrInvalidExperiment, experiment.Name, experiment.Control)
		}
	}
	return experiments, nil
}

// An arm of each experiment flipped per clip, in the order they are listed, so the same dice put a clip in the same
// arms.
func (e Experiments) Assign(dice Dice) Variants {
	return e.assign(dice, false)
}

// An arm of each experiment flipped once a run, before its stories are chosen.
func (e Experiments) AssignRun(dice Dice) Variants {
	return e.assign(dice, true)
}

func (e Experiments) assign(dice Dice, perRun bool) Variants {
	variants := make(Variants, len(e))
	for _, experiment := range e {
		if experiment.PerRun == perRun {
			variants[experiment.Name] = experiment.flip(dice)
		}
	}
	return variants
}

// A clip in an experiment that does not run, or in an arm it does not have, would be counted wrong: it is an error. A
// clip in none of them, made by hand, is not.
func (e Experiments) Check(variants Variants) error {
	for _, name := range slices.Sorted(maps.Keys(variants)) {
		at := slices.IndexFunc(e, func(experiment Experiment) bool { return experiment.Name == name })
		if at < 0 {
			return fmt.Errorf("%w: no experiment %q runs", ErrUnknownVariant, name)
		}
		if !e[at].arm(variants[name]) {
			return fmt.Errorf("%w: the experiment %q has no arm %q", ErrUnknownVariant, name, variants[name])
		}
	}
	return nil
}

// What a run renders: a story, and the arm of each experiment its clip is in.
type Assignment struct {
	Pick     Pick
	Variants Variants
	// The editor's headline, for the generator's --headline; empty for the generator's own.
	Headline string
}

// run: the arms of the run, which every clip carries beside its own. headlines: the editor's, by pick.
func (e Experiments) AssignEach(chosen Shortlist, run Variants, headlines map[Pick]string, dice Dice) []Assignment {
	assignments := make([]Assignment, 0, len(chosen))
	for _, story := range chosen {
		variants := e.Assign(dice)
		maps.Copy(variants, run)
		assignments = append(assignments, Assignment{Pick: story.Pick, Variants: variants, Headline: headlines[story.Pick]})
	}
	return assignments
}
