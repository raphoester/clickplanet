package inmemory_tile_storage

import (
	"encoding/binary"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Every tile's owner and shields, the codes they are written in, and how many tiles each country holds.
type board struct {
	tiles []tileState
	codes *codebook
	held  []uint32
	dirty dirtySet
}

func newBoard(last uint32) *board {
	return &board{
		tiles: make([]tileState, int(last)+1),
		codes: newCodebook(),
		dirty: newDirtySet(int(last) + 1),
	}
}

func (b *board) last() uint32 {
	return uint32(len(b.tiles) - 1) //nolint:gosec // sized from a uint32.
}

func (b *board) intern(country string) (uint16, error) {
	return b.codes.intern(country)
}

func (b *board) codeOf(country string) (uint16, bool) {
	return b.codes.idOf(country)
}

func (b *board) nameOf(code uint16) string {
	return b.codes.nameOf(code)
}

func (b *board) countries() int {
	return b.codes.countries()
}

func (b *board) markDirty(tile uint32) {
	b.dirty.mark(tile)
}

func (b *board) ownerOf(tile uint32) string {
	return b.codes.nameOf(b.tiles[tile].owner)
}

func (b *board) shieldsOn(tile uint32) int {
	return int(b.tiles[tile].shields)
}

// A change of owner takes the tile's shields with it.
func (b *board) move(tile uint32, to uint16) (from uint16) {
	from = b.tiles[tile].owner
	if from != unownedCode {
		b.held[from]--
	}
	b.count(to)

	b.tiles[tile] = ownedBy(to)
	b.dirty.mark(tile)

	return from
}

// A stored tile put back at boot: nothing to flush.
func (b *board) restore(tile uint32, state tileState) {
	b.tiles[tile] = state
	b.count(state.owner)
}

func (b *board) raise(tile uint32) int {
	b.tiles[tile].shields++
	b.dirty.mark(tile)
	return int(b.tiles[tile].shields)
}

func (b *board) strike(tile uint32) int {
	b.tiles[tile].shields--
	b.dirty.mark(tile)
	return int(b.tiles[tile].shields)
}

func (b *board) heldBy(country string) uint32 {
	id, ok := b.codeOf(country)
	if !ok || id == unownedCode || int(id) >= len(b.held) {
		return 0
	}
	return b.held[id]
}

func (b *board) holdings() map[string]uint32 {
	held := map[string]uint32{}
	for id, count := range b.held {
		if id != int(unownedCode) && count > 0 {
			held[b.codes.nameOf(uint16(id))] = count //nolint:gosec // an index of the code table.
		}
	}
	return held
}

func (b *board) batch(start, end uint32) clicks.DenseBatch {
	tiles := make([]byte, 0, (uint64(end-start)+1)*2)
	var shields []clicks.TileShields
	for i, state := range b.tiles[start : uint64(end)+1] {
		tiles = binary.LittleEndian.AppendUint16(tiles, state.owner)
		if state.shields > 0 {
			tile := start + uint32(i) //nolint:gosec // tile <= end, which is a uint32.
			shields = append(shields, clicks.TileShields{Tile: tile, Shields: int(state.shields)})
		}
	}

	return clicks.DenseBatch{Start: start, Codes: b.codes.all(), Tiles: tiles, Shields: shields}
}

func (b *board) drainDirty() []Tile {
	var tiles []Tile
	b.dirty.drain(func(tile uint32) {
		state := b.tiles[tile]
		tiles = append(tiles, Tile{ID: tile, Owner: b.codes.nameOf(state.owner), Shields: int(state.shields)})
	})
	return tiles
}

func (b *board) count(owner uint16) {
	if owner == unownedCode {
		return
	}
	if int(owner) >= len(b.held) {
		b.held = append(b.held, make([]uint32, int(owner)+1-len(b.held))...)
	}
	b.held[owner]++
}
