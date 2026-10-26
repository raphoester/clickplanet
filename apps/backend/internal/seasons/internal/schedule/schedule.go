package schedule

import (
	"bytes"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

// Schedule is every entry the game may play: the seasons' phases at priority 0, and the operators' above them.
type Schedule struct {
	entries []Entry
}

// New is the seasons' phases, each playing the set of its stage. A phase is no operator's entry: its id is zero.
func New(phases []calendar.Phase, sets PhaseSets) (Schedule, error) {
	entries := make([]Entry, 0, len(phases))
	for _, phase := range phases {
		set, err := sets.SetOf(phase.Stage)
		if err != nil {
			return Schedule{}, err
		}
		entries = append(entries, Entry{set: set, startsAt: phase.StartsAt, endsAt: phase.EndsAt})
	}
	return Schedule{entries: entries}, nil
}

func (s Schedule) With(entries ...Entry) Schedule {
	return Schedule{entries: append(append(make([]Entry, 0, len(s.entries)+len(entries)), s.entries...), entries...)}
}

// Turn is what the schedule plays at a time: the running entry with the highest priority, a tie going to the entry
// made last, and default when none runs. Two phases in a row that play one set are one turn.
type Turn struct {
	entry EntryID
	set   SetName
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
		return Turn{set: Default}
	}
	return Turn{entry: playing.id, set: playing.set}
}

// SinceAt is when the turn at a time began: the last start or end before it that changed the turn. The zero time is
// before anything.
func (s Schedule) SinceAt(at time.Time) time.Time {
	turn := s.TurnAt(at)
	var since time.Time
	for _, entry := range s.entries {
		for _, edge := range []time.Time{entry.startsAt, entry.endsAt} {
			if edge.IsZero() || edge.After(at) || !edge.After(since) {
				continue
			}
			if s.TurnAt(edge.Add(-time.Nanosecond)) != turn {
				since = edge
			}
		}
	}
	return since
}

// A zero end never comes.
func (e Entry) runs(at time.Time) bool {
	return !at.Before(e.startsAt) && (e.endsAt.IsZero() || at.Before(e.endsAt))
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
	return TurnSnapshot{Entry: t.entry, Set: t.set}
}

// TurnSnapshot names the operator's entry that plays: the zero id for a phase, or when none runs.
type TurnSnapshot struct {
	Entry EntryID
	Set   SetName
}
