package schedule

import (
	"bytes"
	"cmp"
	"slices"
	"time"
)

// Schedule is every entry the game may play, and the seasons' finales.
type Schedule struct {
	entries []Entry
	finales []Finale
}

func New(entries ...Entry) Schedule {
	return Schedule{entries: entries}
}

// Finale is a season's last battle: its set plays over its window above every entry, whatever the entry's priority.
type Finale struct {
	Set         SetName
	Description Description
	StartsAt    time.Time
	EndsAt      time.Time
}

// With is the schedule with the seasons' finales.
func (s Schedule) With(finales ...Finale) Schedule {
	return Schedule{entries: s.entries, finales: append(slices.Clone(s.finales), finales...)}
}

// Turn is what the schedule plays at a time: the running entry with the highest priority, a tie going to the entry
// made last, and default, a plain turn, when none runs.
type Turn struct {
	entry       EntryID
	set         SetName
	kind        Kind
	description Description
	// startsAt and endsAt are the playing entry's window, in UTC so that two turns compare: zero when none runs.
	startsAt time.Time
	endsAt   time.Time
	finale   bool
}

func (s Schedule) TurnAt(at time.Time) Turn {
	for _, finale := range s.finales {
		if !at.Before(finale.StartsAt) && at.Before(finale.EndsAt) {
			return Turn{
				set: finale.Set, kind: Battle, description: finale.Description, startsAt: finale.StartsAt.UTC(),
				endsAt: finale.EndsAt.UTC(), finale: true,
			}
		}
	}
	var (
		playing Entry
		found   bool
	)
	for _, entry := range s.entries {
		if entry.runs(at) && (!found || entry.outranks(playing)) {
			playing, found = entry, true
		}
	}
	if !found {
		return Turn{set: Default, kind: Plain}
	}
	return Turn{
		entry: playing.id, set: playing.set, kind: playing.kind, description: playing.description,
		startsAt: playing.startsAt.UTC(), endsAt: playing.endsAt.UTC(),
	}
}

// SinceAt is when the turn at a time began: the last start or end before it that changed the turn. The zero time is
// before any entry.
func (s Schedule) SinceAt(at time.Time) time.Time {
	turn := s.TurnAt(at)
	var since time.Time
	for _, edge := range s.edges() {
		if edge.After(at) || !edge.After(since) {
			continue
		}
		if s.TurnAt(edge.Add(-time.Nanosecond)) != turn {
			since = edge
		}
	}
	return since
}

func (s Schedule) edges() []time.Time {
	edges := make([]time.Time, 0, 2*(len(s.entries)+len(s.finales)))
	for _, entry := range s.entries {
		edges = append(edges, entry.startsAt, entry.endsAt)
	}
	for _, finale := range s.finales {
		edges = append(edges, finale.StartsAt, finale.EndsAt)
	}
	return edges
}

// BattlesAfter are the battles that have not started at a time, by start, a tie going to the entry made first. A plain
// entry is the operator's alone.
func (s Schedule) BattlesAfter(at time.Time) []EntrySnapshot {
	var battles []EntrySnapshot
	for _, entry := range s.entries {
		if entry.kind == Battle && entry.startsAt.After(at) {
			battles = append(battles, entry.TakeSnapshot())
		}
	}
	slices.SortFunc(battles, func(a, b EntrySnapshot) int {
		return cmp.Or(a.StartsAt.Compare(b.StartsAt), a.MadeAt.Compare(b.MadeAt), bytes.Compare(a.ID[:], b.ID[:]))
	})
	return battles
}

func (e Entry) runs(at time.Time) bool {
	return !at.Before(e.startsAt) && at.Before(e.endsAt)
}

func (e Entry) outranks(other Entry) bool {
	switch {
	case e.priority != other.priority:
		return e.priority > other.priority
	case !e.madeAt.Equal(other.madeAt):
		return e.madeAt.After(other.madeAt)
	}
	return bytes.Compare(e.id[:], other.id[:]) > 0
}

func (t Turn) TakeSnapshot() TurnSnapshot {
	return TurnSnapshot{
		Entry: t.entry, Set: t.set, Kind: t.kind, Description: t.description, StartsAt: t.startsAt, EndsAt: t.endsAt,
		Finale: t.finale,
	}
}

// TurnSnapshot names the entry that plays: the zero id and no window when none runs, and default plays. A finale
// plays with the zero id and its window.
type TurnSnapshot struct {
	Entry       EntryID
	Set         SetName
	Kind        Kind
	Description Description
	StartsAt    time.Time
	EndsAt      time.Time
	Finale      bool
}
