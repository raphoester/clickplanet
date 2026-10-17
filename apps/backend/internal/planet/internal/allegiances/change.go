package allegiances

import (
	"errors"
	"fmt"
	"time"
)

const changeEvery = time.Hour

var ErrChangedTooSoon = errors.New("the flag was changed too recently to change again")

type Refusal struct {
	Flag  string
	Until time.Time
}

func (r Refusal) Error() string {
	return fmt.Sprintf("%s: it plays for %q until %s", ErrChangedTooSoon, r.Flag, r.Until.UTC().Format(time.RFC3339))
}

func (r Refusal) Unwrap() error { return ErrChangedTooSoon }

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
