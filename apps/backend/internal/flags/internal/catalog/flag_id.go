package catalog

import (
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
)

var ErrInvalidFlagID = errors.New("not the id of a flag sold")

const (
	FlagIDPrefix = "r-"
	FlagIDLength = 10
)

// FlagID names a flag sold, for good: drawn at random, or the code a flag kept when it left the build (xb, gb-eng).
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
