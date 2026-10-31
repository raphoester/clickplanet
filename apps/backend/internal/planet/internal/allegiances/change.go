package allegiances

import (
	"errors"
	"fmt"
	"time"
)

const changeEvery = time.Hour

var (
	ErrChangedTooSoon = errors.New("the flag was changed too recently to change again")
	ErrLocked         = errors.New("the rules in force let no account change its flag")
	ErrOtherFlag      = errors.New("the account plays for another flag")
	ErrNoFlag         = errors.New("the account plays for no flag yet")
	ErrNoAccount      = errors.New("only an account plays for a flag")
)

type Refusal struct {
	Flag   string
	Until  time.Time
	Locked bool
	Reason error
}

func (r Refusal) Error() string {
	if r.Flag == "" {
		return r.Reason.Error()
	}
	return fmt.Sprintf("%s: it plays for %q, changeable at %s", r.Reason, r.Flag, r.Until.UTC().Format(time.RFC3339))
}

func (r Refusal) Unwrap() error { return r.Reason }

type Change struct {
	at time.Time
}

func ChangedAt(at time.Time) Change {
	return Change{at: at}
}

func (c Change) ChangeableAt() time.Time {
	if c.at.IsZero() {
		return time.Time{}
	}
	return c.at.Add(changeEvery)
}
