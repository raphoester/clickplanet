package catalog

import (
	"errors"
	"fmt"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
)

var ErrNameTaken = errors.New("another flag has this name")

// Names is every name a player reads for a flag, the game's own and the ones opened since, open or closed: two flags
// never share one, whatever its case. A removed flag's name is free for a new flag.
type Names struct {
	taken map[string]FlagID
}

func NamesOf(flags []Flag) Names {
	builtIn := cpflags.BuiltIn()
	taken := make(map[string]FlagID, len(builtIn)+len(flags))
	for _, flag := range builtIn {
		taken[strings.ToLower(flag.Name)] = ""
	}
	for _, flag := range flags {
		if flag.state != Removed {
			taken[strings.ToLower(flag.name.String())] = flag.id
		}
	}
	return Names{taken: taken}
}

// Free refuses a name another flag has. A flag's own name is free to it: none, for a flag still to open.
func (n Names) Free(name Name, flag FlagID) error {
	holder, taken := n.taken[strings.ToLower(name.String())]
	if taken && (holder == "" || holder != flag) {
		return fmt.Errorf("%w: %q", ErrNameTaken, name)
	}
	return nil
}
