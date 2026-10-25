package rules

import "time"

// Layers is what the game was asked to play: a baseline, and an operator's override over it while it runs.
type Layers struct {
	baseline Baseline
	override Override
}

func LayersOf(baseline Baseline, override Override) Layers {
	return Layers{baseline: baseline, override: override}
}

// SetAt is the override's set while it runs, then the baseline's, then Default when no baseline was applied.
func (l Layers) SetAt(now time.Time) SetName {
	switch {
	case l.override.runningAt(now):
		return l.override.set
	case l.baseline.set != "":
		return l.baseline.set
	default:
		return Default
	}
}
