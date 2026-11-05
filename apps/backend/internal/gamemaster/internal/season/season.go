package season

import (
	"errors"
	"fmt"
	"time"
)

// Number counts the seasons up from 0.
type Number uint32

// Origin is where season 0's break counts from: the day its first day was scored (#360), so its days keep their numbers.
var Origin = time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)

var (
	ErrInvalidSeason = errors.New("this season cannot be scheduled")
	ErrUnknownSeason = errors.New("no season has this number")
	ErrStarted       = errors.New("this season has started")
	ErrEnded         = errors.New("this season has ended")
	ErrNotEnded      = errors.New("no season of this number has ended")
	ErrNotLast       = errors.New("only the last season is deleted")
)

// Draft is how an operator plans a season: the break after the season before, then its length, and its finale.
type Draft struct {
	Break  time.Duration
	Length time.Duration
	Finale Finale
}

func (d Draft) check() error {
	if d.Break < 0 {
		return refused("its break is %s, below none", d.Break)
	}
	if d.Length <= 0 {
		return refused("its length is %s, not above none", d.Length)
	}
	return d.Finale.check(d.Length)
}

// Season is kept as it was planned: its window follows from the seasons before it.
type Season struct {
	number Number
	brk    time.Duration
	length time.Duration
	finale Finale
}

func (s Season) TakeSnapshot() Snapshot {
	return Snapshot{Number: s.number, Break: s.brk, Length: s.length, Finale: s.finale}
}

type Snapshot struct {
	Number Number
	Break  time.Duration
	Length time.Duration
	Finale Finale
}

func (s Snapshot) Restore() Season {
	return Season{number: s.Number, brk: s.Break, length: s.Length, finale: s.Finale}
}

// Window is a season's start and end, which the calendar works out, and its finale.
type Window struct {
	Number   Number
	StartsAt time.Time
	EndsAt   time.Time
	Finale   Finale
}

func (w Window) Runs(at time.Time) bool {
	return !at.Before(w.StartsAt) && at.Before(w.EndsAt)
}

func (w Window) Started(at time.Time) bool {
	return !at.Before(w.StartsAt)
}

func (w Window) Ended(at time.Time) bool {
	return !at.Before(w.EndsAt)
}

func refused(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalidSeason}, args...)...)
}

func stamp(at time.Time) string { return at.UTC().Format(time.RFC3339) }

// StartsOnBlankMap is true for every season after season 0: its start resets the map.
func (w Window) StartsOnBlankMap() bool {
	return w.Number > 0
}
