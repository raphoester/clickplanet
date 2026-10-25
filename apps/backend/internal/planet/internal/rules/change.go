package rules

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf8"
)

type RequestID string

const maxRequestLength = 128

type Source string

const (
	SourceInternal Source = "internal"
	SourceOperator Source = "operator"
)

var (
	ErrNoRequest      = errors.New("a rule set is applied for a request id, and none was given")
	ErrLongRequest    = fmt.Errorf("a request id is %d characters at most", maxRequestLength)
	ErrAppliedAlready = errors.New("this request was applied already")
)

type Change struct {
	request RequestID
	set     SetName
	source  Source
	at      time.Time
}

// NewChange refuses a request the store already kept a change for: kept is that change, or none.
func NewChange(request RequestID, set Set, source Source, at time.Time, kept Change) (Change, error) {
	switch {
	case kept != (Change{}):
		return kept, fmt.Errorf("%q: %w", request, ErrAppliedAlready)
	case request == "":
		return Change{}, ErrNoRequest
	case utf8.RuneCountInString(string(request)) > maxRequestLength:
		return Change{}, ErrLongRequest
	}

	return Change{request: request, set: set.Name, source: source, at: at}, nil
}

func (c Change) TakeSnapshot() Snapshot {
	return Snapshot{Request: c.request, Set: c.set, Source: c.source, AppliedAt: c.at}
}

type Snapshot struct {
	Request   RequestID
	Set       SetName
	Source    Source
	AppliedAt time.Time
}

func (s Snapshot) Restore() Change {
	return Change{request: s.Request, set: s.Set, source: s.Source, at: s.AppliedAt}
}
