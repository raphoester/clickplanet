package nations

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
)

var ErrUnknownFlag = errors.New("no flag has this code")

type Flag string

var flags = cpcountries.New()

func FlagOf(code string) (Flag, error) {
	if !flags.CheckCountry(code) {
		return "", fmt.Errorf("%w: %q", ErrUnknownFlag, code)
	}
	return Flag(code), nil
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
