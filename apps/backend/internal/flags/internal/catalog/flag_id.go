package catalog

import (
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
)

var ErrInvalidFlagID = errors.New("not the id of a flag opened since the game shipped")

const (
	FlagIDPrefix = "r-"
	FlagIDLength = 10
)

// FlagID names a flag opened since the game shipped, for good: never its name, which a review may change.
type FlagID string

func ParseFlagID(s string) (FlagID, error) {
	if !cpflags.IsOpenedCode(s) {
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
