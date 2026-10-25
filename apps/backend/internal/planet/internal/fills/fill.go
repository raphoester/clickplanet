package fills

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

const maxRequestLength = 128

type Source string

const (
	SourceInternal Source = "internal"
	SourceOperator Source = "operator"
)

var (
	ErrNoRequest      = errors.New("a fill is started for a request id, and none was given")
	ErrLongRequest    = fmt.Errorf("a request id is %d characters at most", maxRequestLength)
	ErrStartedAlready = errors.New("this request started a fill already")
	ErrInvalidFill    = errors.New("not a fill")
)

type Fill struct {
	id         bonuses.FillID
	source     Source
	startedAt  time.Time
	reached    bonuses.Holder
	filled     int
	finishedAt time.Time
}

// NewFill refuses a request the store already kept a fill for: kept is that fill, or none.
func NewFill(request bonuses.FillID, source Source, at time.Time, kept Fill) (Fill, error) {
	if kept != (Fill{}) {
		return kept, fmt.Errorf("%q: %w", request, ErrStartedAlready)
	}

	return Snapshot{ID: request, Source: source, StartedAt: at}.Restore()
}

func (f Fill) ID() bonuses.FillID {
	return f.id
}

// Page is who to fill next: the players before the fill started, past the last one it filled.
type Page struct {
	Before time.Time
	After  bonuses.Holder
	Size   int
}

func (f Fill) NextPage(size int) Page {
	return Page{Before: f.startedAt, After: f.reached, Size: size}
}

// Past is the fill once holders, the page's players, are filled: finished at when the page was not full.
func (f Fill) Past(page Page, holders []bonuses.Holder, at time.Time) Fill {
	if len(holders) > 0 {
		f.reached = holders[len(holders)-1]
		f.filled += len(holders)
	}
	if len(holders) < page.Size {
		f.finishedAt = at
	}

	return f
}

func (f Fill) Running() bool {
	return f.finishedAt.IsZero()
}

func (f Fill) TakeSnapshot() Snapshot {
	return Snapshot{
		ID:         f.id,
		Source:     f.source,
		StartedAt:  f.startedAt,
		Reached:    f.reached,
		Filled:     f.filled,
		FinishedAt: f.finishedAt,
	}
}

type Snapshot struct {
	ID         bonuses.FillID
	Source     Source
	StartedAt  time.Time
	Reached    bonuses.Holder
	Filled     int
	FinishedAt time.Time
}

func (s Snapshot) Restore() (Fill, error) {
	switch {
	case s.ID == "":
		return Fill{}, ErrNoRequest
	case utf8.RuneCountInString(string(s.ID)) > maxRequestLength:
		return Fill{}, ErrLongRequest
	case s.Source != SourceInternal && s.Source != SourceOperator:
		return Fill{}, fmt.Errorf("%w: from %q", ErrInvalidFill, s.Source)
	case s.StartedAt.IsZero() || s.Filled < 0:
		return Fill{}, fmt.Errorf("%w: started at %s, %d filled", ErrInvalidFill, s.StartedAt, s.Filled)
	}

	return Fill{
		id:         s.ID,
		source:     s.Source,
		startedAt:  s.StartedAt,
		reached:    s.Reached,
		filled:     s.Filled,
		finishedAt: s.FinishedAt,
	}, nil
}
