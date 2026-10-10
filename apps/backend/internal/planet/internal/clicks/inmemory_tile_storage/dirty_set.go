package inmemory_tile_storage

import "math/bits"

// One bit per id: what changed since the last flush.
type dirtySet []uint64

func newDirtySet(ids int) dirtySet {
	return make(dirtySet, (ids+63)/64)
}

func (d dirtySet) mark(id uint32) {
	d[id/64] |= 1 << (id % 64)
}

func (d dirtySet) drain(visit func(id uint32)) {
	for w, word := range d {
		if word == 0 {
			continue
		}
		d[w] = 0
		for word != 0 {
			visit(uint32(w*64 + bits.TrailingZeros64(word))) //nolint:gosec // an id the set was sized for.
			word &= word - 1
		}
	}
}
