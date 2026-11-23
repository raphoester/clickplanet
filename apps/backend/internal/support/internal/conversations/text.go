package conversations

import (
	"strings"
	"unicode"
)

const MaxReplyTextLength = 10_000

// A player's words are kept whatever they are: cleaned of control characters, and cut past the length.
func ReplyTextOf(value string) string {
	var b strings.Builder
	for _, r := range strings.NewReplacer("\r\n", "\n", "\r", "\n", "\t", " ").Replace(strings.ToValidUTF8(value, "�")) {
		if unicode.IsControl(r) && r != '\n' {
			continue
		}
		b.WriteRune(r)
	}

	cleaned := []rune(strings.TrimSpace(b.String()))
	if len(cleaned) > MaxReplyTextLength {
		cleaned = cleaned[:MaxReplyTextLength]
	}
	return string(cleaned)
}
