package cpflags

import "regexp"

// A flag the game ships with is an ISO code or gb- and three letters: r- and one letter is neither.
var openedCode = regexp.MustCompile(`^r-[a-z0-9]{10}$`)

// IsOpenedCode is whether a code has the shape of a flag opened since the game shipped. Only planet's catalog knows
// whether one is open.
func IsOpenedCode(code string) bool {
	return openedCode.MatchString(code)
}

// Codes checks a flag code by its shape alone: one the game ships with, or the shape of one opened since. A module
// that only shows or filters by a flag needs no more; planet, which plays it, keeps a catalog.
type Codes struct{}

func (Codes) CheckCountry(code string) bool {
	return IsBuiltIn(code) || IsOpenedCode(code)
}
