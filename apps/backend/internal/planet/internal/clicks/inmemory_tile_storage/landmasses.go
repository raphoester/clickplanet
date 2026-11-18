package inmemory_tile_storage

import "github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"

// How many tiles of each landmass each flag holds.
type landmasses struct {
	borders *clicks.Borders
	held    counts
}

func newLandmasses(borders *clicks.Borders) *landmasses {
	return &landmasses{borders: borders, held: newCounts(borders.Landmasses())}
}

func (l *landmasses) of(tile uint32) clicks.LandmassID {
	return l.borders.LandmassOf(tile)
}

func (l *landmasses) tilesOf(landmass clicks.LandmassID) []uint32 {
	return l.borders.TilesOf(landmass)
}

func (l *landmasses) groundOf(tile uint32) string {
	return l.borders.CountryOf(tile)
}

func (l *landmasses) moved(tile uint32, from, to uint16) {
	l.held.moved(uint32(l.of(tile)), from, to)
}

// Whole: a flag it does not list holds none of the landmass.
func (l *landmasses) row(landmass clicks.LandmassID, nameOf func(flag uint16) string) clicks.LandmassTiles {
	return clicks.LandmassTiles{Landmass: landmass, Flags: l.held.flags(uint32(landmass), nameOf)}
}

func (l *landmasses) each(visit func(landmass clicks.LandmassID)) {
	for landmass := 1; landmass < l.borders.Landmasses(); landmass++ {
		visit(clicks.LandmassID(landmass)) //nolint:gosec // landmasses are uint16 in the blob.
	}
}
