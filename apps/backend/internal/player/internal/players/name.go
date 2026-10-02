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
	// MinNameLength and MaxNameLength count code points after NFC, as postgres' char_length does.
	MinNameLength = 3
	MaxNameLength = 15
	// ReservedPrefix starts no username, in any case: the chat puts it before every guest's name, so a guest
	// cannot pass for a player.
	ReservedPrefix = "guest_"
	// maxMarksInARow is enough marks for a syllable of any script, too few to stack up over other lines.
	maxMarksInARow = 3
)

var (
	ErrInvalidName = errors.New("invalid name")
	// ErrNotLinked is a guest asking for a username: only an account signed in with a provider may hold one.
	ErrNotLinked = errors.New("only an account signed in with a provider may choose a username")
)

// Name is a username, as the player typed it, in NFC. Two names with the same Folded are the same username.
type Name string

// NameOf puts value in NFC and cuts the spaces at its ends, then checks it: MinNameLength to MaxNameLength
// letters of any script, marks after a letter, decimal digits, underscores and single spaces, of one script
// (UTS #39 highly restrictive), not starting with ReservedPrefix once folded. Anything else is refused.
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
		case unicode.In(r, unicode.Other_Default_Ignorable_Code_Point, unicode.Variation_Selector):
			// Hangul fillers are letters and variation selectors marks, but both are invisible.
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

// restrictiveMixes are the sets of scripts UTS #39's highly restrictive profile lets one name mix.
var restrictiveMixes = [][]string{
	{"Latin", "Han", "Hiragana", "Katakana"},
	{"Latin", "Han", "Bopomofo"},
	{"Latin", "Han", "Hangul"},
}

// oneScript is whether value is of one script or one restrictive mix. Common and Inherited belong to all.
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

// scriptOf is the script of r, or "" for one every script shares.
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

// Folded is what two names are compared on, NFKC_Casefold: "Straße" is "STRASSE", "Ａｄａ" is "ada". The store
// keeps it beside the name, since postgres' lower() depends on the locale and folds neither.
func (n Name) Folded() string {
	return norm.NFKC.String(cases.Fold().String(norm.NFKD.String(string(n))))
}

// Profile is what a player chose to be called. A player with no profile has no name.
type Profile struct {
	Account   AccountID
	Name      Name
	UpdatedAt time.Time
	// Admin is an admin of the game. The game only reads it: an operator sets it in the database, and saving a
	// profile never changes it.
	Admin bool
}
