package notes

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxTextLength = 1000

var ErrInvalidText = errors.New("the text is invalid")

func TextOf(value string) (string, error) {
	if !utf8.ValidString(value) {
		return "", fmt.Errorf("%w: not valid UTF-8", ErrInvalidText)
	}

	var b strings.Builder
	for _, r := range strings.NewReplacer("\r\n", "\n", "\r", "\n", "\t", " ").Replace(value) {
		if unicode.IsControl(r) && r != '\n' {
			continue
		}
		b.WriteRune(r)
	}

	cleaned := strings.TrimSpace(b.String())
	if cleaned == "" {
		return "", fmt.Errorf("%w: empty", ErrInvalidText)
	}
	if utf8.RuneCountInString(cleaned) > MaxTextLength {
		return "", fmt.Errorf("%w: longer than %d characters", ErrInvalidText, MaxTextLength)
	}

	return cleaned, nil
}
