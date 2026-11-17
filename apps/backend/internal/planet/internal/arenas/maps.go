package arenas

import (
	"time"
)

// Stretch is the time an arena's map played instead of the world's: from its open to its close, or on while it is open.
// It tells a replay's reader where the world's map stood aside; which map an event was made on is the event's own
// column, never this time.
type Stretch struct {
	ID       ID
	OpenedAt time.Time
	ClosedAt time.Time
	Closed   bool
}

// Overlaps is true for a stretch that played some of a window.
func (s Stretch) Overlaps(since, until time.Time) bool {
	return s.OpenedAt.Before(until) && (!s.Closed || s.ClosedAt.After(since))
}

// Maps is when each arena's map played instead of the world's.
type Maps struct {
	stretches []Stretch
}

// MapsOf reads the stretches of every arena that opened.
func MapsOf(stretches []Stretch) Maps {
	return Maps{stretches: stretches}
}

// Asides are the stretches an arena's map played during a window, instead of the world's.
func (m Maps) Asides(since, until time.Time) []Stretch {
	var asides []Stretch
	for _, stretch := range m.stretches {
		if stretch.Overlaps(since, until) {
			asides = append(asides, stretch)
		}
	}
	return asides
}
