package rules

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf8"
)

type RequestID string

const maxRequestLength = 128

var (
	ErrNoRequest      = errors.New("a rule set is applied for a request id, and none was given")
	ErrLongRequest    = fmt.Errorf("a request id is %d characters at most", maxRequestLength)
	ErrAppliedAlready = errors.New("this request was applied already")
)

// Baseline is the set the game plays while no override runs, until another set is applied.
type Baseline struct {
	request RequestID
	set     SetName
	at      time.Time
}

// NewBaseline refuses a request the store already kept a baseline for: kept is that baseline, or none.
func NewBaseline(request RequestID, set Set, at time.Time, kept Baseline) (Baseline, error) {
	switch {
	case kept != (Baseline{}):
		return kept, fmt.Errorf("%q: %w", request, ErrAppliedAlready)
	case request == "":
		return Baseline{}, ErrNoRequest
	case utf8.RuneCountInString(string(request)) > maxRequestLength:
		return Baseline{}, ErrLongRequest
	}

	return Baseline{request: request, set: set.name, at: at}, nil
}

func (b Baseline) TakeSnapshot() BaselineSnapshot {
	return BaselineSnapshot{Request: b.request, Set: b.set, AppliedAt: b.at}
}

type BaselineSnapshot struct {
	Request   RequestID
	Set       SetName
	AppliedAt time.Time
}

func (s BaselineSnapshot) Restore() Baseline {
	return Baseline{request: s.Request, set: s.Set, at: s.AppliedAt}
}
