package catalog

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

var ErrInvalidNote = errors.New("an early close has a note of 1 to 500 characters")

const maxNoteLength = 500

// Note is why the operator closed a flag before its end, in its own words: the flag's founder reads it, as the EU's
// rules for online platforms ask.
type Note string

func ParseNote(s string) (Note, error) {
	note := strings.TrimSpace(s)
	if !utf8.ValidString(note) || note == "" || utf8.RuneCountInString(note) > maxNoteLength {
		return "", fmt.Errorf("%w: %d characters", ErrInvalidNote, utf8.RuneCountInString(note))
	}
	return Note(note), nil
}

func (n Note) String() string {
	return string(n)
}
