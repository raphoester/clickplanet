package schedule

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

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

// The higher plays.
type Priority uint32

// Kind is what an entry is to the players, who never hear a set's name.
type Kind string

const (
	Plain  Kind = "plain"
	Battle Kind = "battle"
)

// Description is the entry's words for the players: what its rules are, in plain text.
type Description string

const MaxDescription = 300

// Times is how many times a day's points a battle scores, 1 to 10. A plain entry scores none: 0.
type Times uint32

const MaxTimes Times = 10

var (
	ErrInvalidEntry    = errors.New("this entry cannot be scheduled")
	ErrStarted         = errors.New("this entry has started, and runs to its end: schedule another at a higher priority")
	ErrUnknownEntry    = errors.New("no entry has this id")
	ErrUnknownSet      = errors.New("keeps no rule set of this name")
	ErrUnknownKind     = errors.New("no entry is of this kind: plain or battle")
	ErrLongDescription = errors.New("the description is too long")
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

// Draft is what an operator asks the schedule to play.
type Draft struct {
	Set         SetName
	Kind        Kind
	Description Description
	StartsAt    time.Time
	EndsAt      time.Time
	Priority    Priority
	Times       Times
}

// Every reason at once: only an entry to come, with an end, is scheduled.
func (d Draft) check(now time.Time) error {
	if err := d.Kind.check(); err != nil {
		return err
	}
	var found reasons
	if err := d.Set.Check(); err != nil {
		found = append(found, err)
	}
	if err := d.Description.Check(); err != nil {
		found = append(found, err)
	}
	if err := d.Times.Check(d.Kind); err != nil {
		found = append(found, err)
	}
	if !d.StartsAt.After(now) {
		found = append(found, fmt.Errorf("it starts at %s, not after now", d.StartsAt.UTC().Format(time.RFC3339)))
	}
	if !d.EndsAt.After(d.StartsAt) {
		found = append(found, fmt.Errorf("it ends at %s, not after its start", d.EndsAt.UTC().Format(time.RFC3339)))
	}
	if len(found) > 0 {
		return fmt.Errorf("%w: %w", ErrInvalidEntry, found)
	}
	return nil
}

// reasons reads as one sentence, and errors.Is finds each.
type reasons []error

func (r reasons) Error() string {
	words := make([]string, 0, len(r))
	for _, reason := range r {
		words = append(words, reason.Error())
	}
	return strings.Join(words, "; ")
}

func (r reasons) Unwrap() []error { return r }

func (n SetName) Check() error {
	if !namePattern.MatchString(string(n)) {
		return fmt.Errorf("%q is no rule set's name: a lowercase letter, then up to 31 of a-z, 0-9, - and _", n)
	}
	return nil
}

func (d Description) Check() error {
	if n := utf8.RuneCountInString(string(d)); n > MaxDescription {
		return fmt.Errorf("%w: %d characters, at most %d", ErrLongDescription, n, MaxDescription)
	}
	return nil
}

func (t Times) Check(kind Kind) error {
	switch kind {
	case Battle:
		if t < 1 || t > MaxTimes {
			return fmt.Errorf("a battle scores %d times a day's points, not 1 to %d", t, MaxTimes)
		}
	case Plain:
		if t != 0 {
			return fmt.Errorf("a plain entry scores no points, not %d times", t)
		}
	}
	return nil
}

func (k Kind) check() error {
	switch k {
	case Plain, Battle:
		return nil
	}
	return fmt.Errorf("%w: %q", ErrUnknownKind, k)
}

type Entry struct {
	id          EntryID
	set         SetName
	kind        Kind
	description Description
	startsAt    time.Time
	endsAt      time.Time
	priority    Priority
	times       Times
	madeAt      time.Time
}

func NewEntry(id EntryID, draft Draft, now time.Time) (Entry, error) {
	if err := draft.check(now); err != nil {
		return Entry{}, err
	}
	return Entry{
		id: id, set: draft.Set, kind: draft.Kind, description: draft.Description, startsAt: draft.StartsAt, endsAt: draft.EndsAt,
		priority: draft.Priority, times: draft.Times, madeAt: now,
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
		id: e.id, set: draft.Set, kind: draft.Kind, description: draft.Description, startsAt: draft.StartsAt, endsAt: draft.EndsAt,
		priority: draft.Priority, times: draft.Times, madeAt: e.madeAt,
	}, nil
}

func (e Entry) TakeSnapshot() EntrySnapshot {
	return EntrySnapshot{
		ID: e.id, Set: e.set, Kind: e.kind, Description: e.description, StartsAt: e.startsAt, EndsAt: e.endsAt,
		Priority: e.priority, Times: e.times, MadeAt: e.madeAt,
	}
}

type EntrySnapshot struct {
	ID          EntryID
	Set         SetName
	Kind        Kind
	Description Description
	StartsAt    time.Time
	EndsAt      time.Time
	Priority    Priority
	Times       Times
	MadeAt      time.Time
}

func (s EntrySnapshot) Restore() Entry {
	return Entry{
		id: s.ID, set: s.Set, kind: s.Kind, description: s.Description, startsAt: s.StartsAt, endsAt: s.EndsAt,
		priority: s.Priority, times: s.Times, madeAt: s.MadeAt,
	}
}
