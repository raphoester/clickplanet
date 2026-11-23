package catalog

import (
	"errors"
	"fmt"
	"regexp"
)

var ErrInvalidFlagID = errors.New("not the id of a flag opened since the game shipped")

// A flag the game ships with is an ISO code or gb- and three letters: r- and one letter is neither.
const (
	FlagIDPrefix = "r-"
	FlagIDLength = 10
)

var flagIDPattern = regexp.MustCompile(`^r-[a-z0-9]{10}$`)

// FlagID names a flag opened since the game shipped, for good: never its name, which a review may change.
type FlagID string

func ParseFlagID(s string) (FlagID, error) {
	if !flagIDPattern.MatchString(s) {
		return "", fmt.Errorf("%w: %q", ErrInvalidFlagID, s)
	}
	return FlagID(s), nil
}

func (id FlagID) String() string {
	return string(id)
}

type IDProvider interface {
	NewID() (FlagID, error)
}
