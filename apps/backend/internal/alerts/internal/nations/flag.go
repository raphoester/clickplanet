package nations

import (
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
)

type Flag string

// FlagOf is the flag of a role in the server: none for a flag opened since the game shipped, which has no role yet.
func FlagOf(code string) (Flag, bool) {
	if !cpflags.IsBuiltIn(code) {
		return "", false
	}
	return Flag(code), true
}

var invented = map[Flag]string{"xb": "Brittany"}

func (f Flag) Name() string {
	if name, found := invented[f]; found {
		return name
	}
	region, err := language.ParseRegion(string(f))
	if err != nil {
		return strings.ToUpper(string(f))
	}
	if name := display.English.Regions().Name(region); name != "" {
		return name
	}
	return strings.ToUpper(string(f))
}

// Discord draws the two regional indicator letters of an ISO code as its flag.
func (f Flag) Emoji() string {
	if _, found := invented[f]; found {
		return "🏴"
	}
	var emoji strings.Builder
	for _, letter := range strings.ToUpper(string(f)) {
		emoji.WriteRune('🇦' + (letter - 'A'))
	}
	return emoji.String()
}
