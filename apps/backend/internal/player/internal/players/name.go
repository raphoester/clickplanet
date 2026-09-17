package players

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	MinNameLength = 3
	MaxNameLength = 20
	// ReservedPrefix starts no username, in any case: the chat puts it before every guest's name, so a guest
	// cannot pass for a player.
	ReservedPrefix = "guest_"
)

var (
	ErrInvalidName = errors.New("invalid name")
	// ErrNotLinked is a guest asking for a username: only an account signed in with a provider may hold one.
	ErrNotLinked = errors.New("only an account signed in with a provider may choose a username")
)

// Name is a username, as the player typed it. Two names that differ only in case are the same username.
type Name string

// NameOf checks a username: MinNameLength to MaxNameLength characters, each an ASCII letter, a digit or an
// underscore, and not starting with ReservedPrefix in any case. Nothing is cleaned: a name that breaks a rule
// is refused, never changed into one the player did not type.
func NameOf(value string) (Name, error) {
	if len(value) < MinNameLength || len(value) > MaxNameLength {
		return "", fmt.Errorf("%w: not %d to %d characters", ErrInvalidName, MinNameLength, MaxNameLength)
	}
	for _, r := range value {
		if !usernameRune(r) {
			return "", fmt.Errorf("%w: %q is not a letter, a digit or an underscore", ErrInvalidName, r)
		}
	}
	if strings.HasPrefix(strings.ToLower(value), ReservedPrefix) {
		return "", fmt.Errorf("%w: starts with %q", ErrInvalidName, ReservedPrefix)
	}

	return Name(value), nil
}

func usernameRune(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_'
}

// Folded is the name in lower case: what two names are compared on.
func (n Name) Folded() string {
	return strings.ToLower(string(n))
}

// Profile is what a player chose to be called. A player with no profile has no name.
type Profile struct {
	Account   AccountID
	Name      Name
	UpdatedAt time.Time
}
