package messages

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const defaultMaxTextLength = 280

// Limits bound a text in runes.
type Limits struct {
	maxText int
}

// NewLimits takes zero or less as the default.
func NewLimits(maxText int) Limits {
	if maxText <= 0 {
		maxText = defaultMaxTextLength
	}
	return Limits{maxText: maxText}
}

func (l Limits) Text(value string) (string, error) {
	text, err := clean(value, l.maxText)
	if err != nil {
		return "", fmt.Errorf("%w: text: %w", ErrInvalidMessage, err)
	}
	return text, nil
}

func clean(value string, maxLength int) (string, error) {
	if !utf8.ValidString(value) {
		return "", fmt.Errorf("not valid UTF-8")
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
		return "", fmt.Errorf("empty")
	}

	if utf8.RuneCountInString(cleaned) > maxLength {
		return "", fmt.Errorf("longer than %d characters", maxLength)
	}

	return cleaned, nil
}
