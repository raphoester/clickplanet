package catalog

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

var ErrInvalidName = errors.New("a flag's name is 1 to 40 characters, none of them a control character")

const maxNameLength = 40

// Name is what the players read for a flag opened since the game shipped, in English.
type Name string

func ParseName(s string) (Name, error) {
	name := strings.TrimSpace(s)
	if !utf8.ValidString(name) || name == "" || utf8.RuneCountInString(name) > maxNameLength ||
		strings.ContainsFunc(name, unicode.IsControl) {
		return "", fmt.Errorf("%w: %q", ErrInvalidName, s)
	}
	return Name(name), nil
}

func (n Name) String() string {
	return string(n)
}
