package players

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// MaxNameLength is in runes, as the chat counts a name.
const MaxNameLength = 24

var ErrInvalidName = errors.New("invalid name")

// Name is a name a player chose, already cleaned.
type Name string

// NameOf cleans a name the way the chat does: valid UTF-8, a tab is a space, control characters are removed,
// the ends are trimmed, and what is left is not empty and at most MaxNameLength runes.
func NameOf(value string) (Name, error) {
	if !utf8.ValidString(value) {
		return "", fmt.Errorf("%w: not valid UTF-8", ErrInvalidName)
	}

	var b strings.Builder
	for _, r := range value {
		if r == '\t' {
			r = ' '
		}
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}

	cleaned := strings.TrimSpace(b.String())
	if cleaned == "" {
		return "", fmt.Errorf("%w: empty", ErrInvalidName)
	}
	if utf8.RuneCountInString(cleaned) > MaxNameLength {
		return "", fmt.Errorf("%w: longer than %d characters", ErrInvalidName, MaxNameLength)
	}

	return Name(cleaned), nil
}

// Profile is what a player chose to be called. A player with no profile has no name.
type Profile struct {
	Account   AccountID
	Name      Name
	UpdatedAt time.Time
}
