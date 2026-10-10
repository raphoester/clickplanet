package inmemory_tile_storage

import "github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"

// How many tiles of each landmass each flag holds, and the flag each landmass is locked to.
type landmasses struct {
	borders *clicks.Borders
	held    [][]uint32
	locks   []uint16
	dirty   dirtySet
}

func newLandmasses(borders *clicks.Borders) *landmasses {
	return &landmasses{
		borders: borders,
		held:    make([][]uint32, borders.Landmasses()),
		locks:   make([]uint16, borders.Landmasses()),
		dirty:   newDirtySet(borders.Landmasses()),
	}
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

func (l *landmasses) asset() string {
	return l.borders.Asset()
}

func (l *landmasses) known(landmass clicks.LandmassID) bool {
	return int(landmass) < len(l.locks)
}

func (l *landmasses) moved(tile uint32, from, to uint16) {
	landmass := l.of(tile)
	if from != unownedCode {
		*l.count(landmass, from)--
	}
	if to != unownedCode {
		*l.count(landmass, to)++
	}
}

func (l *landmasses) heldBy(landmass clicks.LandmassID, flag uint16) int {
	held := l.held[landmass]
	if int(flag) >= len(held) {
		return 0
	}
	return int(held[flag])
}

func (l *landmasses) lockedTo(landmass clicks.LandmassID) uint16 {
	return l.locks[landmass]
}

func (l *landmasses) lock(landmass clicks.LandmassID, flag uint16) {
	l.locks[landmass] = flag
	l.dirty.mark(uint32(landmass))
}

func (l *landmasses) markDirty(landmass clicks.LandmassID) {
	l.dirty.mark(uint32(landmass))
}

// A stored lock put back at boot: nothing to flush.
func (l *landmasses) restore(landmass clicks.LandmassID, flag uint16) {
	l.locks[landmass] = flag
}

func (l *landmasses) each(visit func(landmass clicks.LandmassID)) {
	for landmass := 1; landmass < len(l.locks); landmass++ {
		visit(clicks.LandmassID(landmass)) //nolint:gosec // landmasses are uint16 in the blob.
	}
}

func (l *landmasses) drainDirty(nameOf func(flag uint16) string) []Landmass {
	var dirty []Landmass
	l.dirty.drain(func(id uint32) {
		landmass := clicks.LandmassID(id) //nolint:gosec // landmasses are uint16 in the blob.
		dirty = append(dirty, Landmass{ID: landmass, Asset: l.borders.Asset(), FortifiedBy: nameOf(l.locks[landmass])})
	})
	return dirty
}

func (l *landmasses) count(landmass clicks.LandmassID, flag uint16) *uint32 {
	held := l.held[landmass]
	if int(flag) >= len(held) {
		held = append(held, make([]uint32, int(flag)+1-len(held))...)
		l.held[landmass] = held
	}
	return &held[flag]
}
