package schedule

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type EntryID uuid.UUID

func ParseEntryID(id string) (EntryID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return EntryID{}, fmt.Errorf("%w: %q", ErrUnknownEntry, id)
	}
	return EntryID(parsed), nil
}

func (id EntryID) String() string { return uuid.UUID(id).String() }

// SetName names a rule set as planet and the antibot name theirs.
type SetName string

// Default is what plays when no entry runs.
const Default SetName = "default"

// Priority 0 is the seasons' phases; an operator's entry is above it.
type Priority uint32

var (
	ErrInvalidEntry = errors.New("this entry cannot be scheduled")
	ErrStarted      = errors.New("this entry has started, and runs to its end: schedule another at a higher priority")
	ErrUnknownEntry = errors.New("no entry has this id")
	ErrUnknownSet   = errors.New("keeps no rule set of this name")
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

// Draft is what an operator asks the schedule to play.
type Draft struct {
	Set      SetName
	StartsAt time.Time
	EndsAt   time.Time
	Priority Priority
}

// Every reason at once: only an entry to come, above the phases, with an end, is scheduled.
func (d Draft) check(now time.Time) error {
	var reasons []string
	if err := d.Set.check(); err != nil {
		reasons = append(reasons, err.Error())
	}
	if d.Priority == 0 {
		reasons = append(reasons, "priority 0 is the seasons' phases: an entry is above it")
	}
	if !d.StartsAt.After(now) {
		reasons = append(reasons, fmt.Sprintf("it starts at %s, not after now", d.StartsAt.UTC().Format(time.RFC3339)))
	}
	if !d.EndsAt.After(d.StartsAt) {
		reasons = append(reasons, fmt.Sprintf("it ends at %s, not after its start", d.EndsAt.UTC().Format(time.RFC3339)))
	}
	if len(reasons) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalidEntry, strings.Join(reasons, "; "))
	}
	return nil
}

func (n SetName) check() error {
	if !namePattern.MatchString(string(n)) {
		return fmt.Errorf("%q is no rule set's name: a lowercase letter, then up to 31 of a-z, 0-9, - and _", n)
	}
	return nil
}

type Entry struct {
	id       EntryID
	set      SetName
	startsAt time.Time
	endsAt   time.Time
	priority Priority
	madeAt   time.Time
}

func NewEntry(id EntryID, draft Draft, now time.Time) (Entry, error) {
	if err := draft.check(now); err != nil {
		return Entry{}, err
	}
	return Entry{
		id: id, set: draft.Set, startsAt: draft.StartsAt, endsAt: draft.EndsAt, priority: draft.Priority, madeAt: now,
	}, nil
}

// Changeable is nil until the entry starts: only an entry to come is changed or deleted.
func (e Entry) Changeable(now time.Time) error {
	if !now.Before(e.startsAt) {
		return fmt.Errorf("%s, since %s: %w", e.id, e.startsAt.UTC().Format(time.RFC3339), ErrStarted)
	}
	return nil
}

// Edited is the entry with the draft's values, and keeps its place in a tie.
func (e Entry) Edited(draft Draft, now time.Time) (Entry, error) {
	if err := e.Changeable(now); err != nil {
		return Entry{}, err
	}
	if err := draft.check(now); err != nil {
		return Entry{}, err
	}
	return Entry{
		id: e.id, set: draft.Set, startsAt: draft.StartsAt, endsAt: draft.EndsAt, priority: draft.Priority, madeAt: e.madeAt,
	}, nil
}

func (e Entry) TakeSnapshot() EntrySnapshot {
	return EntrySnapshot{
		ID: e.id, Set: e.set, StartsAt: e.startsAt, EndsAt: e.endsAt, Priority: e.priority, MadeAt: e.madeAt,
	}
}

type EntrySnapshot struct {
	ID       EntryID
	Set      SetName
	StartsAt time.Time
	EndsAt   time.Time
	Priority Priority
	MadeAt   time.Time
}

func (s EntrySnapshot) Restore() Entry {
	return Entry{id: s.ID, set: s.Set, startsAt: s.StartsAt, endsAt: s.EndsAt, priority: s.Priority, madeAt: s.MadeAt}
}
