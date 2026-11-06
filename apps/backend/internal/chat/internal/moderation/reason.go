package moderation

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxReasonLength = 500

var ErrInvalidReason = errors.New("the reason is invalid")

// Why an operator acted, in a few words. Optional: NoReason is none.
type Reason string

const NoReason Reason = ""

// Support's text rule (notes.TextOf), but empty is no reason: lines are kept, every other control character is
// stripped, and the ends are trimmed.
func ReasonOf(value string) (Reason, error) {
	if !utf8.ValidString(value) {
		return NoReason, fmt.Errorf("%w: not valid UTF-8", ErrInvalidReason)
	}

	var b strings.Builder
	for _, r := range strings.NewReplacer("\r\n", "\n", "\r", "\n", "\t", " ").Replace(value) {
		if unicode.IsControl(r) && r != '\n' {
			continue
		}
		b.WriteRune(r)
	}

	cleaned := strings.TrimSpace(b.String())
	if utf8.RuneCountInString(cleaned) > MaxReasonLength {
		return NoReason, fmt.Errorf("%w: longer than %d characters", ErrInvalidReason, MaxReasonLength)
	}
	return Reason(cleaned), nil
}
