package season

import (
	"errors"
	"fmt"
	"time"
)

// Number counts the seasons up from 0.
type Number uint32

// Origin is where season 0's break counts from: the day its countdown shipped, before scoring counted any take.
var Origin = time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)

var (
	ErrInvalidSeason = errors.New("this season cannot be scheduled")
	ErrUnknownSeason = errors.New("no season has this number")
	ErrStarted       = errors.New("this season has started")
	ErrEnded         = errors.New("this season has ended")
	ErrNotLast       = errors.New("only the last season is deleted")
)

// Draft is how an operator plans a season: the break after the season before, then its length.
type Draft struct {
	Break  time.Duration
	Length time.Duration
}

func (d Draft) check() error {
	if d.Break < 0 {
		return refused("its break is %s, below none", d.Break)
	}
	if d.Length <= 0 {
		return refused("its length is %s, not above none", d.Length)
	}
	return nil
}

// Season is kept as it was planned: its window follows from the seasons before it.
type Season struct {
	number Number
	brk    time.Duration
	length time.Duration
}

func (s Season) TakeSnapshot() Snapshot {
	return Snapshot{Number: s.number, Break: s.brk, Length: s.length}
}

type Snapshot struct {
	Number Number
	Break  time.Duration
	Length time.Duration
}

func (s Snapshot) Restore() Season {
	return Season{number: s.Number, brk: s.Break, length: s.Length}
}

// Window is a season's start and end, which the calendar works out.
type Window struct {
	Number   Number
	StartsAt time.Time
	EndsAt   time.Time
}

func (w Window) runs(at time.Time) bool {
	return !at.Before(w.StartsAt) && at.Before(w.EndsAt)
}

func (w Window) started(at time.Time) bool {
	return !at.Before(w.StartsAt)
}

func (w Window) ended(at time.Time) bool {
	return !at.Before(w.EndsAt)
}

func refused(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalidSeason}, args...)...)
}

func stamp(at time.Time) string { return at.UTC().Format(time.RFC3339) }
