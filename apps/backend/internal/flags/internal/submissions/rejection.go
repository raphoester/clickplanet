package submissions

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

var ErrInvalidRejection = errors.New("a rejection has a reason, and a note of 500 characters at most that other needs")

const maxNoteLength = 500

// Reason is why the operator refused a submission. Its player reads it, as the EU's rules for online platforms ask.
type Reason string

const (
	NotAPlace  Reason = "not_a_place"
	Exists     Reason = "exists"
	Unreadable Reason = "unreadable"
	Hateful    Reason = "hateful"
	Other      Reason = "other"
)

// Rejection is the reason and the operator's own words, which the player reads too.
type Rejection struct {
	Reason Reason
	Note   string
}

func ParseRejection(reason Reason, note string) (Rejection, error) {
	rejection := Rejection{Reason: reason, Note: strings.TrimSpace(note)}
	return rejection, rejection.check()
}

func (r Rejection) check() error {
	switch r.Reason {
	case NotAPlace, Exists, Unreadable, Hateful, Other:
	default:
		return fmt.Errorf("%w: the reason %q", ErrInvalidRejection, r.Reason)
	}
	switch {
	case !utf8.ValidString(r.Note) || utf8.RuneCountInString(r.Note) > maxNoteLength:
		return fmt.Errorf("%w: a note of %d characters", ErrInvalidRejection, utf8.RuneCountInString(r.Note))
	case r.Reason == Other && r.Note == "":
		return fmt.Errorf("%w: other with no note", ErrInvalidRejection)
	}
	return nil
}
