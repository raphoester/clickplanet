package players

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

const (
	MinNameLength  = 3
	MaxNameLength  = 15
	ReservedPrefix = "guest_"
	maxMarksInARow = 3
)

var (
	ErrInvalidName = errors.New("invalid name")
	ErrNotLinked   = errors.New("only an account signed in with a provider may choose a username")
)

type Name string

func NameOf(value string) (Name, error) {
	value = strings.Trim(norm.NFC.String(value), " ")

	length := utf8.RuneCountInString(value)
	if !utf8.ValidString(value) || length < MinNameLength || length > MaxNameLength {
		return "", fmt.Errorf("%w: not %d to %d characters", ErrInvalidName, MinNameLength, MaxNameLength)
	}
	if strings.Contains(value, "  ") {
		return "", fmt.Errorf("%w: two spaces in a row", ErrInvalidName)
	}
	if err := checkRunes(value); err != nil {
		return "", err
	}
	if !oneScript(value) {
		return "", fmt.Errorf("%w: mixes scripts", ErrInvalidName)
	}

	name := Name(value)
	if strings.HasPrefix(name.Folded(), ReservedPrefix) {
		return "", fmt.Errorf("%w: starts with %q", ErrInvalidName, ReservedPrefix)
	}
	return name, nil
}

func checkRunes(value string) error {
	marks := 0
	previous := ' '
	for _, r := range value {
		switch {
		// Before the letter case: Hangul fillers are letters, yet invisible.
		case unicode.In(r, unicode.Other_Default_Ignorable_Code_Point, unicode.Variation_Selector):
			return fmt.Errorf("%w: %q is invisible", ErrInvalidName, r)
		case unicode.IsLetter(r), unicode.Is(unicode.Nd, r), r == '_', r == ' ':
			marks = 0
		case unicode.In(r, unicode.Mn, unicode.Mc) && (unicode.IsLetter(previous) || unicode.IsMark(previous)) &&
			marks < maxMarksInARow:
			marks++
		default:
			return fmt.Errorf("%w: %q is not a letter, a digit, an underscore or a space", ErrInvalidName, r)
		}
		previous = r
	}
	return nil
}

var restrictiveMixes = [][]string{
	{"Latin", "Han", "Hiragana", "Katakana"},
	{"Latin", "Han", "Bopomofo"},
	{"Latin", "Han", "Hangul"},
}

func oneScript(value string) bool {
	seen := cpcolls.NewSet[string]()
	for _, r := range value {
		if script := scriptOf(r); script != "" {
			seen.Add(script)
		}
	}
	if seen.Len() <= 1 {
		return true
	}

	for _, mix := range restrictiveMixes {
		inMix := 0
		for _, script := range mix {
			if seen.Contains(script) {
				inMix++
			}
		}
		if inMix == seen.Len() {
			return true
		}
	}
	return false
}

func scriptOf(r rune) string {
	if unicode.In(r, unicode.Common, unicode.Inherited) {
		return ""
	}
	for script, table := range unicode.Scripts {
		if unicode.Is(table, r) {
			return script
		}
	}
	return ""
}

func (n Name) Folded() string {
	return norm.NFKC.String(cases.Fold().String(norm.NFKD.String(string(n))))
}

type Profile struct {
	Account   AccountID
	Name      Name
	UpdatedAt time.Time
	Admin     bool
	Color     Color
}
