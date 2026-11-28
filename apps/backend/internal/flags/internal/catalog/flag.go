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
	ErrUnknownFlag   = errors.New("no flag was opened with this id")
	ErrRemoved       = errors.New("the operator removed the flag: nobody pays for it")
	ErrEnded         = errors.New("a flag's end is in the future")
	ErrHeld          = errors.New("another account holds the open flag: only its holder adds time")
	ErrOrderedBefore = errors.New("the flag's time ended after the order: an order placed before claims nothing")
	ErrInvalidDays   = errors.New("a flag gets 1 to 3660 days at a time")
)

const maxDays = 3660

type State string

const (
	Open State = "open"
	// Its time ended, at its end or by a refund: anyone may pay for it, and so claim it.
	Closed State = "closed"
	// The operator removed it, for good: nobody pays for it, and its name is free for a new flag.
	Removed State = "removed"
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
	// The account whose payment opened it or claimed it last: none for a flag opened by hand.
	holder AccountID
	// Only while removed.
	note Note
	// The zero Anthem for none.
	anthem Anthem
}

func NewOpening(ids IDProvider, clock cptime.Clock) Opening {
	return Opening{ids: ids, clock: clock}
}

type Opening struct {
	ids   IDProvider
	clock cptime.Clock
}

// Open is a flag opened by hand, which no account holds, with no anthem.
func (o Opening) Open(name Name, picture pictures.Picture, pin Pin, endsAt time.Time) (Flag, error) {
	return o.OpenFor(AccountID{}, name, picture, pin, Anthem{}, endsAt)
}

// OpenFor is a flag its holder's payment opens, with the anthem its submission chose.
func (o Opening) OpenFor(
	holder AccountID, name Name, picture pictures.Picture, pin Pin, anthem Anthem, endsAt time.Time,
) (Flag, error) {
	now := o.clock.Now()
	if !endsAt.After(now) {
		return Flag{}, fmt.Errorf("%w: %s", ErrEnded, endsAt.Format(time.RFC3339))
	}
	id, err := o.ids.NewID()
	if err != nil {
		return Flag{}, fmt.Errorf("failed to draw a flag id: %w", err)
	}
	return Flag{
		id: id, name: name, picture: picture, pin: pin, state: Open, decidedAt: now, endsAt: endsAt, holder: holder,
		anthem: anthem,
	}, nil
}

// TimeAdded is the operator's time: an open flag ends a span later, a closed one opens again from now, its holder
// kept. A removed one comes back too, its note dropped, unless a new flag took its name.
func (f Flag) TimeAdded(span Span, names Names, now time.Time) (Flag, error) {
	switch f.state {
	case Open:
		f.endsAt = span.After(f.endsAt)
		return f, nil
	case Removed:
		if err := names.Free(f.name, f.id); err != nil {
			return f, err //nolint:wrapcheck // the names say which.
		}
		return f.reopened(span, now), nil
	case Closed:
		return f.reopened(span, now), nil
	}
	return f, fmt.Errorf("the flag %s is %q, which is no state", f.id, f.state)
}

func (f Flag) reopened(span Span, now time.Time) Flag {
	f.state, f.decidedAt, f.endsAt, f.note = Open, now, span.After(now), ""
	return f
}

// PayableBy is nil when the account may pay for the flag now: its holder, or anyone once its time ended, which the
// payment claims.
func (f Flag) PayableBy(account AccountID) error {
	switch {
	case f.state == Removed:
		return fmt.Errorf("%s: %w", f.id, ErrRemoved)
	case f.holder == account, f.state == Closed:
		return nil
	}
	return fmt.Errorf("%s: %w", f.id, ErrHeld)
}

// Paid is the flag with a payment's time, and how much later it ends. The holder's open flag ends a span later, and
// opens again from now once closed. Another account's payment claims a closed flag, open from now and held by it,
// only when its order was placed since the flag closed: an older order, a late renewal of a plan bought before, claims
// nothing.
func (f Flag) Paid(by AccountID, orderedAt time.Time, span Span, now time.Time) (Flag, time.Duration, error) {
	if err := f.PayableBy(by); err != nil {
		return f, 0, err
	}
	switch {
	case f.state == Open:
		end := span.After(f.endsAt)
		added := end.Sub(f.endsAt)
		f.endsAt = end
		return f, added, nil
	case f.holder != by && orderedAt.Before(f.decidedAt):
		return f, 0, fmt.Errorf("%s closed %s, ordered %s: %w", f.id, f.decidedAt.Format(time.RFC3339),
			orderedAt.Format(time.RFC3339), ErrOrderedBefore)
	}
	f = f.reopened(span, now)
	f.holder = by
	return f, f.endsAt.Sub(now), nil
}

// Shortened ends the flag that much sooner, and closes it now when that end is past. Only time granted since the flag
// last opened is taken back: an earlier opening's time was spent, by its holder then. A closed flag stays as it is.
func (f Flag) Shortened(by time.Duration, grantedAt, now time.Time) Flag {
	if f.state != Open || grantedAt.Before(f.decidedAt) {
		return f
	}
	f.endsAt = f.endsAt.Add(-by)
	if !f.endsAt.After(now) {
		f.state, f.decidedAt = Closed, now
	}
	return f
}

// Removed is the flag the operator removed, open or closed, with the note its holder reads.
func (f Flag) Removed(note Note, at time.Time) (Flag, error) {
	if f.state == Removed {
		return f, fmt.Errorf("%s: %w", f.id, ErrRemoved)
	}
	if note == "" {
		return f, fmt.Errorf("%s: %w", f.id, ErrInvalidNote)
	}
	f.state, f.decidedAt, f.note = Removed, at, note
	return f, nil
}

func (f Flag) Renamed(name Name, at time.Time) (Flag, *flagsv1.FlagRenamed) {
	f.name = name
	return f, &flagsv1.FlagRenamed{Flag: f.id.String(), Name: name.String(), RenamedAt: timestamppb.New(at)}
}

// Sung is the flag with the anthem its holder chose, or none: open or closed (a claim keeps it), never once removed.
// False when it plays that anthem already, and nothing is told.
func (f Flag) Sung(by AccountID, anthem Anthem, at time.Time) (Flag, *flagsv1.AnthemChanged, error) {
	switch {
	case f.state == Removed:
		return f, nil, fmt.Errorf("%s: %w", f.id, ErrRemoved)
	case f.holder == AccountID{} || f.holder != by:
		return f, nil, fmt.Errorf("%s: %w", f.id, ErrNotHolder)
	case f.anthem == anthem:
		return f, nil, nil
	}
	f.anthem = anthem
	return f, &flagsv1.AnthemChanged{Flag: f.id.String(), ChangedAt: timestamppb.New(at)}, nil
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
	case Closed, Removed:
		return []proto.Message{&flagsv1.FlagClosed{Flag: f.id.String(), ClosedAt: timestamppb.New(f.decidedAt)}}
	}
	return nil
}

func (f Flag) TakeSnapshot() Snapshot {
	return Snapshot{
		ID: f.id, Name: f.name, Picture: f.picture.TakeSnapshot(), Pin: f.pin, State: f.state, DecidedAt: f.decidedAt,
		EndsAt: f.endsAt, Holder: f.holder, Note: f.note, Anthem: f.anthem.Library(),
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
	// The zero id for none.
	Holder AccountID
	Note   Note
	// The library code of its anthem, empty for none.
	Anthem string
}

func (s Snapshot) Restore() (Flag, error) {
	picture, err := s.Picture.Restore()
	if err != nil {
		return Flag{}, fmt.Errorf("the flag %s: %w", s.ID, err)
	}
	if err := s.Pin.check(); err != nil {
		return Flag{}, fmt.Errorf("the flag %s: %w", s.ID, err)
	}
	anthem, err := ParseAnthem(s.Anthem)
	if err != nil {
		return Flag{}, fmt.Errorf("the flag %s: %w", s.ID, err)
	}
	if (s.Note != "") != (s.State == Removed) {
		return Flag{}, fmt.Errorf("the flag %s is %q with the note %q: only a removed flag has one", s.ID, s.State, s.Note)
	}
	switch s.State {
	case Open, Closed, Removed:
		return Flag{
			id: s.ID, name: s.Name, picture: picture, pin: s.Pin, state: s.State, decidedAt: s.DecidedAt,
			endsAt: s.EndsAt, holder: s.Holder, note: s.Note, anthem: anthem,
		}, nil
	}
	return Flag{}, fmt.Errorf("the flag %s is %q, which is no state", s.ID, s.State)
}
