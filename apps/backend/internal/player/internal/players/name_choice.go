package players

import (
	"errors"
	"fmt"
	"time"
)

const renameEvery = 30 * 24 * time.Hour

var ErrRenamedTooSoon = errors.New("the name was chosen too recently to change")

type RenameRefusal struct {
	Until time.Time
}

func (r RenameRefusal) Error() string {
	return fmt.Sprintf("%s: it may change from %s", ErrRenamedTooSoon, r.Until.UTC().Format(time.RFC3339))
}

func (r RenameRefusal) Unwrap() error { return ErrRenamedTooSoon }

type NameChoice struct {
	at time.Time
}

func NameChosenAt(at time.Time) NameChoice {
	return NameChoice{at: at}
}

func (c NameChoice) RenamableAt() time.Time {
	if c.at.IsZero() {
		return time.Time{}
	}
	return c.at.Add(renameEvery)
}

func (c NameChoice) RenameError(at time.Time) error {
	if until := c.RenamableAt(); at.Before(until) {
		return RenameRefusal{Until: until}
	}
	return nil
}
