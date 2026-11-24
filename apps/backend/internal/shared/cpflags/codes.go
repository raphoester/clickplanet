package cpflags

import "regexp"

// Two letters or gb- and three, as the build's, or r- and ten drawn for a flag opened since.
var flagCode = regexp.MustCompile(`^([a-z]{2}(-[a-z]{3})?|r-[a-z0-9]{10})$`)

// IsOpenedCode is a flag's code the build does not ship: one opened since, or one sold since it left the build (xb).
func IsOpenedCode(code string) bool {
	return flagCode.MatchString(code) && !IsBuiltIn(code)
}

// Codes checks a code by its shape alone; planet, which plays the flags, keeps a catalog.
type Codes struct{}

func (Codes) CheckCountry(code string) bool {
	return flagCode.MatchString(code)
}
