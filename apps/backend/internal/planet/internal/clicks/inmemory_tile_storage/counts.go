package inmemory_tile_storage

import "github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"

// How many tiles of each row each flag holds: a row is a landmass, or a province.
type counts [][]uint32

func newCounts(rows int) counts {
	return make(counts, rows)
}

func (c counts) moved(row uint32, from, to uint16) {
	if from != unownedCode {
		*c.count(row, from)--
	}
	if to != unownedCode {
		*c.count(row, to)++
	}
}

func (c counts) heldBy(row uint32, flag uint16) int {
	held := c[row]
	if int(flag) >= len(held) {
		return 0
	}
	return int(held[flag])
}

// Whole: a flag it does not list holds none of the row.
func (c counts) flags(row uint32, nameOf func(flag uint16) string) []clicks.FlagTiles {
	var flags []clicks.FlagTiles
	for flag, tiles := range c[row] {
		if flag != int(unownedCode) && tiles > 0 {
			flags = append(flags, clicks.FlagTiles{Flag: nameOf(uint16(flag)), Tiles: tiles}) //nolint:gosec // an index of the code table.
		}
	}
	return flags
}

func (c counts) count(row uint32, flag uint16) *uint32 {
	held := c[row]
	if int(flag) >= len(held) {
		held = append(held, make([]uint32, int(flag)+1-len(held))...)
		c[row] = held
	}
	return &held[flag]
}
