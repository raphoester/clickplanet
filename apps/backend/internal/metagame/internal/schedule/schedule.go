package schedule

import (
	"bytes"
	"time"
)

// Schedule is every entry the game may play.
type Schedule struct {
	entries []Entry
}

func New(entries ...Entry) Schedule {
	return Schedule{entries: entries}
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
}

func (s Schedule) TurnAt(at time.Time) Turn {
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
	for _, entry := range s.entries {
		for _, edge := range []time.Time{entry.startsAt, entry.endsAt} {
			if edge.After(at) || !edge.After(since) {
				continue
			}
			if s.TurnAt(edge.Add(-time.Nanosecond)) != turn {
				since = edge
			}
		}
	}
	return since
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
	}
}

// TurnSnapshot names the entry that plays: the zero id and no window when none runs, and default plays.
type TurnSnapshot struct {
	Entry       EntryID
	Set         SetName
	Kind        Kind
	Description Description
	StartsAt    time.Time
	EndsAt      time.Time
}
