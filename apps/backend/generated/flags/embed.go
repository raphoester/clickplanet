// Package flagdata holds the flags the game ships with, copied from the page's static/countries/countries.json by
// `make flags`: the page draws them from that file, and the backend builds from its own directory.
//
// Embedded rather than read from disk, for the reason generated/quiz is: cmd/api is a self-contained container.
package flagdata

import _ "embed"

//go:embed countries.json
var countries []byte

func Countries() []byte { return countries }
