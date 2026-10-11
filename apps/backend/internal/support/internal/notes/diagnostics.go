package notes

import (
	"strings"
	"unicode"
)

const (
	maxBuildLength     = 64
	maxUserAgentLength = 256
	maxViewportSide    = 100_000
)

// What a bug report says of the page and the browser it came from. Plain data, a part of a note's snapshot: the
// page wrote it, so a note keeps it cleaned and trusts none of it.
type Diagnostics struct {
	Build          string
	ViewportWidth  uint32
	ViewportHeight uint32
	ConnectionDown bool
	UserAgent      string
}

func (d Diagnostics) cleaned() Diagnostics {
	return Diagnostics{
		Build:          printable(d.Build, maxBuildLength),
		ViewportWidth:  min(d.ViewportWidth, maxViewportSide),
		ViewportHeight: min(d.ViewportHeight, maxViewportSide),
		ConnectionDown: d.ConnectionDown,
		UserAgent:      printable(d.UserAgent, maxUserAgentLength),
	}
}

func printable(value string, maxRunes int) string {
	cleaned := []rune(strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.ToValidUTF8(value, ""))))

	if len(cleaned) > maxRunes {
		cleaned = cleaned[:maxRunes]
	}
	return strings.TrimSpace(string(cleaned))
}
