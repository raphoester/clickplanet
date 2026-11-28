package catalog

import (
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
)

var (
	ErrInvalidAnthem = errors.New("an anthem from the game's library is the code of a flag the game ships with")
	ErrNotHolder     = errors.New("only the flag's holder chooses its anthem")
)

// Anthem is a sold flag's music, which the game plays as a country's when the flag leads: for now a recording of the
// game's library, the one it plays for a flag it ships with, named by that flag's code. The zero Anthem is none. The
// page offers only the codes it has a recording for; a code with none plays nothing, as it does for its own flag.
type Anthem struct {
	library string
}

// ParseAnthem reads a library code: none for an empty one.
func ParseAnthem(library string) (Anthem, error) {
	if library == "" {
		return Anthem{}, nil
	}
	if !cpflags.IsBuiltIn(library) {
		return Anthem{}, fmt.Errorf("%w: %q", ErrInvalidAnthem, library)
	}
	return Anthem{library: library}, nil
}

// The code of the flag the game ships with whose recording plays, empty for none.
func (a Anthem) Library() string {
	return a.library
}

func (a Anthem) IsZero() bool {
	return a == Anthem{}
}
