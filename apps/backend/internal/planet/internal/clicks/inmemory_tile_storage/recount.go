package inmemory_tile_storage

import (
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// The landmasses and the flags whose counts changed since the last census was published.
type recount struct {
	landmasses dirtySet
	flags      dirtySet
}

func newRecount(landmasses int) recount {
	return recount{landmasses: newDirtySet(landmasses), flags: newDirtySet(math.MaxUint16 + 1)}
}

func (r recount) moved(landmass clicks.LandmassID, from, to uint16) {
	r.landmasses.mark(uint32(landmass))
	r.flags.mark(uint32(from))
	r.flags.mark(uint32(to))
}
