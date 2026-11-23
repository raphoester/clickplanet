package inmemory_tile_storage

import (
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// The traps laid on the map in play, and the tiles whose trap changed since the last flush. A trap sprung leaves the
// map in the claim that sprang it.
type trapBook struct {
	laid    map[uint32]clicks.Trap
	unsaved *cpcolls.Set[uint32]
}

func newTrapBook() *trapBook {
	return &trapBook{laid: map[uint32]clicks.Trap{}, unsaved: cpcolls.NewSet[uint32]()}
}

func (b *trapBook) has(tile uint32) bool {
	_, ok := b.laid[tile]
	return ok
}

func (b *trapBook) lay(trap clicks.Trap) {
	tile := trap.TakeSnapshot().Tile
	b.laid[tile] = trap
	b.unsaved.Add(tile)
}

// A stored trap put back at boot: nothing to flush.
func (b *trapBook) restore(kept clicks.TrapSnapshot) {
	b.laid[kept.Tile] = kept.Restore()
}

// take takes the tile's trap off the map; false for a tile with none.
func (b *trapBook) take(tile uint32) (clicks.Trap, bool) {
	trap, ok := b.laid[tile]
	if !ok {
		return clicks.Trap{}, false
	}
	delete(b.laid, tile)
	b.unsaved.Add(tile)

	return trap, true
}

// forget takes every trap of a deleted account off the map.
func (b *trapBook) forget(account cpsession.AccountID) {
	for tile, trap := range b.laid {
		if trap.TakeSnapshot().Account == account {
			b.take(tile)
		}
	}
}

func (b *trapBook) of(flag string) []uint32 {
	var tiles []uint32
	for tile, trap := range b.laid {
		if trap.TakeSnapshot().Flag == flag {
			tiles = append(tiles, tile)
		}
	}
	slices.Sort(tiles)
	return tiles
}

// drainUnsaved is each trap laid and each tile whose trap left since the last flush.
func (b *trapBook) drainUnsaved() Traps {
	tiles := make([]uint32, 0, b.unsaved.Len())
	b.unsaved.ForEach(func(tile uint32) { tiles = append(tiles, tile) })
	slices.Sort(tiles)
	b.unsaved.Clear()

	var traps Traps
	for _, tile := range tiles {
		if trap, ok := b.laid[tile]; ok {
			traps.Laid = append(traps.Laid, trap.TakeSnapshot())
			continue
		}
		traps.Gone = append(traps.Gone, tile)
	}

	return traps
}

func (b *trapBook) markUnsaved(traps Traps) {
	for _, trap := range traps.Laid {
		b.unsaved.Add(trap.Tile)
	}
	b.unsaved.Add(traps.Gone...)
}

// Traps is what changed among a map's traps: each one laid to keep, and each tile whose trap is to delete.
type Traps struct {
	Laid []clicks.TrapSnapshot
	Gone []uint32
}

func (t Traps) empty() bool {
	return len(t.Laid) == 0 && len(t.Gone) == 0
}

func (s *Storage) TrapsOf(flag string) ([]uint32, clicks.MapVersion) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.traps.of(flag), s.version
}

func (l *insideLaying) Trapped(tile uint32) bool {
	return l.storage.traps.has(tile)
}

func (l *insideLaying) PutTrap(trap clicks.Trap) {
	l.storage.traps.lay(trap)
}
