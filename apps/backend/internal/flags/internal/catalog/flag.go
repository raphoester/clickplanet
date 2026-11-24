package catalog

import (
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	ErrUnknownFlag = errors.New("no flag was opened with this id")
	ErrClosed      = errors.New("the flag is closed already")
	ErrEnded       = errors.New("a flag's end is in the future")
	ErrTooLate     = errors.New("a flag closed 30 days ago or more cannot come back")
	ErrInvalidDays = errors.New("a flag gets 1 to 3660 days at a time")
)

// A flag closed less long ago comes back with the time added to it.
const comeback = 30 * 24 * time.Hour

const maxDays = 3660

type State string

const (
	Open   State = "open"
	Closed State = "closed"
)

type Days uint32

func ParseDays(days uint32) (Days, error) {
	if days == 0 || days > maxDays {
		return 0, fmt.Errorf("%w: %d", ErrInvalidDays, days)
	}
	return Days(days), nil
}

func (d Days) After(t time.Time) time.Time {
	return t.Add(time.Duration(d) * 24 * time.Hour)
}

// Span is time added to a flag: days by hand, or a paid period.
type Span interface {
	After(t time.Time) time.Time
}

// Flag is a flag opened since the game shipped. It is closed, never deleted: history still names it.
type Flag struct {
	id        FlagID
	name      Name
	picture   pictures.Picture
	pin       Pin
	state     State
	decidedAt time.Time
	endsAt    time.Time
	// Only while the operator's close holds: a flag closed at its end has none.
	note Note
}

func NewOpening(ids IDProvider, clock cptime.Clock) Opening {
	return Opening{ids: ids, clock: clock}
}

type Opening struct {
	ids   IDProvider
	clock cptime.Clock
}

func (o Opening) Open(name Name, picture pictures.Picture, pin Pin, endsAt time.Time) (Flag, error) {
	now := o.clock.Now()
	if !endsAt.After(now) {
		return Flag{}, fmt.Errorf("%w: %s", ErrEnded, endsAt.Format(time.RFC3339))
	}
	id, err := o.ids.NewID()
	if err != nil {
		return Flag{}, fmt.Errorf("failed to draw a flag id: %w", err)
	}
	return Flag{id: id, name: name, picture: picture, pin: pin, state: Open, decidedAt: now, endsAt: endsAt}, nil
}

func (f Flag) TimeAdded(span Span, now time.Time) (Flag, error) {
	added, _, err := f.Extended(span, now)
	return added, err
}

// Extended is the flag with the span added, and how much later it ends: an open flag a span after its end, one closed
// less than 30 days ago open again from now.
func (f Flag) Extended(span Span, now time.Time) (Flag, time.Duration, error) {
	if err := f.Payable(now); err != nil {
		return f, 0, err
	}
	switch f.state {
	case Open:
		end := span.After(f.endsAt)
		added := end.Sub(f.endsAt)
		f.endsAt = end
		return f, added, nil
	case Closed:
		f.state, f.decidedAt, f.endsAt, f.note = Open, now, span.After(now), ""
		return f, f.endsAt.Sub(now), nil
	}
	return f, 0, fmt.Errorf("the flag %s is %q, which is no state", f.id, f.state)
}

// Payable is nil while time added would keep the flag or bring it back: open, or closed less than 30 days ago.
func (f Flag) Payable(now time.Time) error {
	if f.state == Closed && now.Sub(f.decidedAt) >= comeback {
		return fmt.Errorf("%s closed %s: %w", f.id, f.decidedAt.Format(time.RFC3339), ErrTooLate)
	}
	return nil
}

// Shortened ends the flag that much sooner, and closes it now when that end is past. A closed flag stays as it is.
func (f Flag) Shortened(by time.Duration, now time.Time) Flag {
	if f.state != Open {
		return f
	}
	f.endsAt = f.endsAt.Add(-by)
	if !f.endsAt.After(now) {
		f.state, f.decidedAt = Closed, now
	}
	return f
}

func (f Flag) Closed(note Note, at time.Time) (Flag, error) {
	if f.state == Closed {
		return f, fmt.Errorf("%s: %w", f.id, ErrClosed)
	}
	if note == "" {
		return f, fmt.Errorf("%s: %w", f.id, ErrInvalidNote)
	}
	f.state, f.decidedAt, f.note = Closed, at, note
	return f, nil
}

func (f Flag) Renamed(name Name, at time.Time) (Flag, *flagsv1.FlagRenamed) {
	f.name = name
	return f, &flagsv1.FlagRenamed{Flag: f.id.String(), Name: name.String(), RenamedAt: timestamppb.New(at)}
}

// Expired is false while the flag has time left, or is closed already.
func (f Flag) Expired(now time.Time) (Flag, bool) {
	if f.state != Open || f.endsAt.After(now) {
		return f, false
	}
	f.state, f.decidedAt = Closed, f.endsAt
	return f, true
}

// Facts is what a module that never heard of the flag must hear to know it as it is now.
func (f Flag) Facts() []proto.Message {
	switch f.state {
	case Open:
		return []proto.Message{&flagsv1.FlagOpened{
			Flag: f.id.String(), Name: f.name.String(), OpenedAt: timestamppb.New(f.decidedAt), Pin: f.pin.message(),
		}}
	case Closed:
		return []proto.Message{&flagsv1.FlagClosed{Flag: f.id.String(), ClosedAt: timestamppb.New(f.decidedAt)}}
	}
	return nil
}

func (f Flag) TakeSnapshot() Snapshot {
	return Snapshot{
		ID: f.id, Name: f.name, Picture: f.picture.TakeSnapshot(), Pin: f.pin, State: f.state, DecidedAt: f.decidedAt,
		EndsAt: f.endsAt, Note: f.note,
	}
}

type Snapshot struct {
	ID        FlagID
	Name      Name
	Picture   pictures.Snapshot
	Pin       Pin
	State     State
	DecidedAt time.Time
	EndsAt    time.Time
	Note      Note
}

func (s Snapshot) Restore() (Flag, error) {
	picture, err := s.Picture.Restore()
	if err != nil {
		return Flag{}, fmt.Errorf("the flag %s: %w", s.ID, err)
	}
	if err := s.Pin.check(); err != nil {
		return Flag{}, fmt.Errorf("the flag %s: %w", s.ID, err)
	}
	if s.Note != "" && s.State != Closed {
		return Flag{}, fmt.Errorf("the flag %s is %q with a close's note", s.ID, s.State)
	}
	switch s.State {
	case Open, Closed:
		return Flag{
			id: s.ID, name: s.Name, picture: picture, pin: s.Pin, state: s.State, decidedAt: s.DecidedAt,
			endsAt: s.EndsAt, note: s.Note,
		}, nil
	}
	return Flag{}, fmt.Errorf("the flag %s is %q, which is no state", s.ID, s.State)
}
