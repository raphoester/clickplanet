package inmemory_tile_storage

import (
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// The landmasses, the provinces and the flags whose counts changed since the last tally was published.
type recount struct {
	landmasses dirtySet
	provinces  dirtySet
	flags      dirtySet
}

func newRecount(landmasses, provinces int) recount {
	return recount{
		landmasses: newDirtySet(landmasses), provinces: newDirtySet(provinces), flags: newDirtySet(math.MaxUint16 + 1),
	}
}

func (r recount) moved(landmass clicks.LandmassID, province clicks.ProvinceID, from, to uint16) {
	r.landmasses.mark(uint32(landmass))
	r.provinces.mark(uint32(province))
	r.flags.mark(uint32(from))
	r.flags.mark(uint32(to))
}
