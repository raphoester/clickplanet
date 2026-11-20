package posts

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// The name of one of the lab's experiments (docs/video-lab.md): lower case, digits and dashes, "far-invasion".
type ExperimentName string

// What the posts of no experiment record: the clips as they are, which every experiment is compared with.
const Control ExperimentName = "control"

var (
	ErrInvalidExperiment = errors.New("the experiment is invalid")

	experimentNames = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
)

func ExperimentNameOf(name string) (ExperimentName, error) {
	if !experimentNames.MatchString(name) || ExperimentName(name) == Control {
		return "", fmt.Errorf("%w: %q is no name of an experiment", ErrInvalidExperiment, name)
	}
	return ExperimentName(name), nil
}

// A story the editor writes to a brief of its own, beside the run's other stories: the prose of what to tell and how.
type Experiment struct {
	Name  ExperimentName
	Brief string
}

// The experiments on. None is no lab: every run is the control's.
type Lab struct {
	Experiments []Experiment
}

func LabOf(experiments ...Experiment) (Lab, error) {
	seen := cpcolls.NewSet[ExperimentName]()
	for _, experiment := range experiments {
		if _, err := ExperimentNameOf(string(experiment.Name)); err != nil {
			return Lab{}, err
		}
		if strings.TrimSpace(experiment.Brief) == "" {
			return Lab{}, fmt.Errorf("%w: %s has no brief", ErrInvalidExperiment, experiment.Name)
		}
		if seen.Contains(experiment.Name) {
			return Lab{}, fmt.Errorf("%w: %s is on twice", ErrInvalidExperiment, experiment.Name)
		}
		seen.Add(experiment.Name)
	}
	return Lab{Experiments: experiments}, nil
}

// The pick of a run its experiment fills, from 1, and the experiment; At 0 is none.
type LabSlot struct {
	At         int
	Experiment Experiment
}

func (s LabSlot) On() bool {
	return s.At > 0
}

// The run's first pick is the lab's 2 times in 3, else its second: TikTok takes the first alone, so 2 of its 3 a day
// are the lab's, and YouTube, which takes both, half. Which experiment is even among those on. Rolled before the
// editor reads anything, so a slot's arm is a coin and never a choice made from the story: the arms compare fair.
// count: the clips the run makes; a slot past them is none.
func (l Lab) Slot(count int, dice Dice) LabSlot {
	if len(l.Experiments) == 0 || count < 1 {
		return LabSlot{}
	}
	at := 1
	if dice.IntN(3) == 2 {
		at = 2
	}
	experiment := l.Experiments[dice.IntN(len(l.Experiments))]
	if at > count {
		return LabSlot{}
	}
	return LabSlot{At: at, Experiment: experiment}
}

// A slot the run's seasonal stories take moves to the first they leave, or is none.
func (s LabSlot) after(first, count int) LabSlot {
	if !s.On() {
		return s
	}
	if s.At <= first {
		s.At = first + 1
	}
	if s.At > count {
		return LabSlot{}
	}
	return s
}

// The hook is drawn as a headline is, over the opening; none is none.
func hookFrom(written string) (string, error) {
	if hook := headlineOf(oneLine(written)); hook != "" {
		return drawn("hook", hook, headlineRunes)
	}
	return "", nil
}

var languages = regexp.MustCompile(`^[a-z]{2,3}$`)

// BCP 47's primary subtag, in lower case: "pl". None is English.
func languageFrom(written string) (string, error) {
	language := strings.ToLower(strings.TrimSpace(written))
	if language == "" {
		return English, nil
	}
	if !languages.MatchString(language) {
		return "", fmt.Errorf("%w: %q is no language", errUndrawable, written)
	}
	return language, nil
}

const English = "en"

func (c Clip) experimentOrControl() ExperimentName {
	if c.experiment == "" {
		return Control
	}
	return c.experiment
}
