package nations

import (
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
)

// A flag the server has a role for: one the game ships with, or one sold since whose name the alerts heard.
type Flag string

// The English name of a flag the game ships with. A sold flag's name is the flags module's: the Catalog keeps it.
func (f Flag) Name() string {
	region, err := language.ParseRegion(string(f))
	if err != nil {
		return strings.ToUpper(string(f))
	}
	if name := display.English.Regions().Name(region); name != "" {
		return name
	}
	return strings.ToUpper(string(f))
}

// Discord draws the two regional indicator letters of an ISO code as its flag, and England's, Scotland's and Wales'
// as the black flag with the tags of their codes. Any other sold flag has none: Brittany's xb would be two letters.
func (f Flag) Emoji() string {
	code := string(f)
	if cpflags.IsBuiltIn(code) {
		var emoji strings.Builder
		for _, letter := range strings.ToUpper(code) {
			emoji.WriteRune('🇦' + (letter - 'A'))
		}
		return emoji.String()
	}
	if nation, found := strings.CutPrefix(code, "gb-"); found && len(nation) == 3 {
		var emoji strings.Builder
		emoji.WriteRune('🏴')
		for _, letter := range "gb" + nation {
			emoji.WriteRune(tag + letter)
		}
		emoji.WriteRune(cancelTag)
		return emoji.String()
	}
	return ""
}

const (
	// The tag letters run beside ASCII, from U+E0000.
	tag       = 0xE0000
	cancelTag = 0xE007F
)

// How a ping names the flag: its emoji when it has one, then its name.
func Label(flag Flag, name string) string {
	if emoji := flag.Emoji(); emoji != "" {
		return emoji + " " + name
	}
	return name
}
