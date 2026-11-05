package messages

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpscripts"
)

const defaultMaxTextLength = 280

type Limits struct {
	maxText int
}

func NewLimits(maxText int) Limits {
	if maxText <= 0 {
		maxText = defaultMaxTextLength
	}
	return Limits{maxText: maxText}
}

// A text or a GIF, never both.
func (l Limits) Body(text string, gif Gif) (string, Gif, error) {
	if gif.None() {
		cleaned, err := l.Text(text)
		return cleaned, Gif{}, err
	}
	if text != "" {
		return "", Gif{}, fmt.Errorf("%w: a GIF with a text", ErrInvalidMessage)
	}
	checked, err := gif.checked()
	if err != nil {
		return "", Gif{}, fmt.Errorf("%w: GIF: %w", ErrInvalidMessage, err)
	}
	return "", checked, nil
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
		if cpscripts.Historic(r) {
			return "", fmt.Errorf("%q is of a historic script", r)
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
