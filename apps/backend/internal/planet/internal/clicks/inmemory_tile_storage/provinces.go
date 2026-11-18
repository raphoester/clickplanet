package inmemory_tile_storage

import "github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"

// How many tiles of each province each flag holds, and the flag each province is locked to: the ground fortify plays
// on.
type provinces struct {
	table *clicks.Provinces
	held  counts
	locks []uint16
	dirty dirtySet
}

func newProvinces(table *clicks.Provinces) *provinces {
	rows := table.Count() + 1
	return &provinces{table: table, held: newCounts(rows), locks: make([]uint16, rows), dirty: newDirtySet(rows)}
}

func (p *provinces) of(tile uint32) clicks.ProvinceID {
	return p.table.ProvinceOf(tile)
}

func (p *provinces) tilesOf(province clicks.ProvinceID) []uint32 {
	return p.table.TilesOf(province)
}

func (p *provinces) asset() string {
	return p.table.Asset()
}

func (p *provinces) known(province clicks.ProvinceID) bool {
	return province != clicks.NoProvince && int(province) < len(p.locks)
}

func (p *provinces) moved(tile uint32, from, to uint16) {
	p.held.moved(uint32(p.of(tile)), from, to)
}

func (p *provinces) heldBy(province clicks.ProvinceID, flag uint16) int {
	return p.held.heldBy(uint32(province), flag)
}

// Whole: a flag it does not list holds none of the province.
func (p *provinces) row(province clicks.ProvinceID, nameOf func(flag uint16) string) clicks.ProvinceTiles {
	return clicks.ProvinceTiles{Province: province, Flags: p.held.flags(uint32(province), nameOf)}
}

func (p *provinces) lockedTo(province clicks.ProvinceID) uint16 {
	return p.locks[province]
}

func (p *provinces) lock(province clicks.ProvinceID, flag uint16) {
	p.locks[province] = flag
	p.dirty.mark(uint32(province))
}

func (p *provinces) markDirty(province clicks.ProvinceID) {
	p.dirty.mark(uint32(province))
}

// A stored lock put back at boot: nothing to flush.
func (p *provinces) restore(province clicks.ProvinceID, flag uint16) {
	p.locks[province] = flag
}

func (p *provinces) each(visit func(province clicks.ProvinceID)) {
	for province := 1; province < len(p.locks); province++ {
		visit(clicks.ProvinceID(province)) //nolint:gosec // provinces are uint16 in the file.
	}
}

func (p *provinces) drainDirty(nameOf func(flag uint16) string) []Province {
	var dirty []Province
	p.dirty.drain(func(id uint32) {
		province := clicks.ProvinceID(id) //nolint:gosec // provinces are uint16 in the file.
		dirty = append(dirty, Province{ID: province, Asset: p.table.Asset(), FortifiedBy: nameOf(p.locks[province])})
	})
	return dirty
}
