package arenas

import (
	"time"
)

// Stretch is the time an arena's map played instead of the world's: from its open to its close, or on while it is open.
type Stretch struct {
	ID       ID
	OpenedAt time.Time
	ClosedAt time.Time
	Closed   bool
}

// Holds is true for a ledger entry made on the arena's map: after its open, and before the second it closed. The ledger
// keeps whole seconds, so a world take right after the close shares that second; the arena's freeze keeps its own takes
// off it.
func (s Stretch) Holds(at time.Time) bool {
	return at.After(s.OpenedAt) && (!s.Closed || at.Before(s.ClosedAt.Truncate(time.Second)))
}

// Before is the time the stretch's entries come before, as SQL compares them: an open stretch's ends at until.
func (s Stretch) Before(until time.Time) time.Time {
	if !s.Closed {
		return until
	}
	return s.ClosedAt.Truncate(time.Second)
}

// Overlaps is true for a stretch that played some of a window.
func (s Stretch) Overlaps(since, until time.Time) bool {
	return s.OpenedAt.Before(until) && (!s.Closed || s.ClosedAt.After(since))
}

// Played is when one map played, as the ledger keeps time: after a time (none: from the start), and outside the
// stretches another map played.
type Played struct {
	After time.Time
	Gaps  []Stretch
}

// Holds is true for a ledger entry made on the map.
func (p Played) Holds(at time.Time) bool {
	if !p.After.IsZero() && !at.After(p.After) {
		return false
	}
	for _, gap := range p.Gaps {
		if gap.Holds(at) {
			return false
		}
	}
	return true
}

// Bounds are the gaps as SQL compares them: an entry after afters[i] and before befores[i] is in one. A gap still open
// ends at until.
func (p Played) Bounds(until time.Time) (afters, befores []time.Time) {
	afters, befores = make([]time.Time, len(p.Gaps)), make([]time.Time, len(p.Gaps))
	for i, gap := range p.Gaps {
		afters[i], befores[i] = gap.OpenedAt, gap.Before(until)
	}
	return afters, befores
}

// Maps is when each map played: the world's, but for each arena's stretch.
type Maps struct {
	stretches []Stretch
}

// MapsOf reads the stretches of every arena that opened.
func MapsOf(stretches []Stretch) Maps {
	return Maps{stretches: stretches}
}

// World is when the world's map played: around every arena's stretch.
func (m Maps) World() Played {
	return Played{Gaps: m.stretches}
}

// Playing is when the map that plays now played: the open arena's since it opened, or the world's.
func (m Maps) Playing() Played {
	if open, ok := m.Open(); ok {
		return Played{After: open.OpenedAt}
	}
	return m.World()
}

// Open is the stretch of the arena open now, if one is.
func (m Maps) Open() (Stretch, bool) {
	for _, stretch := range m.stretches {
		if !stretch.Closed {
			return stretch, true
		}
	}
	return Stretch{}, false
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
