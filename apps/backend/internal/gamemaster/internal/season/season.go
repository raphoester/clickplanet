package season

import (
	"errors"
	"fmt"
	"time"
)

// Number counts the seasons up from 0.
type Number uint32

var (
	ErrInvalidSeason = errors.New("this season cannot be scheduled")
	ErrUnknownSeason = errors.New("no season has this number")
	ErrStarted       = errors.New("this season has started")
	ErrEnded         = errors.New("this season has ended")
	ErrNotLast       = errors.New("only the last season is deleted")
)

// Draft is the window an operator asks a season to run.
type Draft struct {
	StartsAt time.Time
	EndsAt   time.Time
}

type Season struct {
	number   Number
	startsAt time.Time
	endsAt   time.Time
}

func (s Season) runs(at time.Time) bool {
	return !at.Before(s.startsAt) && at.Before(s.endsAt)
}

func (s Season) started(at time.Time) bool {
	return !at.Before(s.startsAt)
}

func (s Season) ended(at time.Time) bool {
	return !at.Before(s.endsAt)
}

func (s Season) TakeSnapshot() Snapshot {
	return Snapshot{Number: s.number, StartsAt: s.startsAt, EndsAt: s.endsAt}
}

type Snapshot struct {
	Number   Number
	StartsAt time.Time
	EndsAt   time.Time
}

func (s Snapshot) Restore() Season {
	return Season{number: s.Number, startsAt: s.StartsAt, endsAt: s.EndsAt}
}

func refused(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalidSeason}, args...)...)
}

func stamp(at time.Time) string { return at.UTC().Format(time.RFC3339) }
