// Package experiments holds the lab's experiments (docs/video-lab.md): each a brief the editor writes one story of a run
// to, in briefs/<name>.md, embedded. A commit turns one on or off.
package experiments

import (
	"embed"
	"fmt"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

//go:embed briefs/*.md
var briefs embed.FS

var (
	// The experiments on, in the order they started.
	On = []string{"far-invasion", "carousel", "war-report"}
	// The ones that ran and stopped, or wait their turn (docs/video-lab.md, "The order"): their briefs stay, so what
	// their posts were written to can still be read, and `clips edit -experiment` still tries them.
	Off = []string{"call-to-arms"}

	// The experiments drawn as something else than a clip of the globe.
	formats = map[string]posts.Format{"carousel": posts.Carousel}
	// The experiments a voice tells, over the globe.
	narrated = []string{"war-report"}
)

// The experiments on, each with its brief.
func Lab() (posts.Lab, error) {
	experiments := make([]posts.Experiment, 0, len(On))
	for _, name := range On {
		experiment, err := Named(name)
		if err != nil {
			return posts.Lab{}, err
		}
		experiments = append(experiments, experiment)
	}
	return posts.LabOf(experiments...) //nolint:wrapcheck // the domain already says what is wrong.
}

// Any experiment there is a brief of, on or off: `clips edit -lab <name>` tries one by hand.
func Named(name string) (posts.Experiment, error) {
	if !slices.Contains(On, name) && !slices.Contains(Off, name) {
		return posts.Experiment{}, fmt.Errorf("%w: no experiment is named %q", posts.ErrInvalidExperiment, name)
	}
	known, err := posts.ExperimentNameOf(name)
	if err != nil {
		return posts.Experiment{}, err //nolint:wrapcheck // the domain already says what is wrong.
	}
	brief, err := briefs.ReadFile("briefs/" + name + ".md")
	if err != nil {
		return posts.Experiment{}, fmt.Errorf("%w: %s has no brief: %w", posts.ErrInvalidExperiment, name, err)
	}
	format, apart := formats[name]
	if !apart {
		format = posts.GlobeReplay
	}
	return posts.Experiment{Name: known, Brief: string(brief), Format: format, Narrated: slices.Contains(narrated, name)}, nil
}
