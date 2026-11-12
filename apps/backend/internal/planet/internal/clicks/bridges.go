package clicks

import "fmt"

// Bridges is each crossing of the sea the map draws, by its two end tiles.
type Bridges struct {
	starts []uint32
	ends   []uint32
}

// NewBridges takes each bridge as its two end tiles.
func NewBridges(tiles uint32, pairs [][2]uint32) (*Bridges, error) {
	sizes := make([]uint32, tiles+2)
	for i, pair := range pairs {
		for _, tile := range pair {
			if tile == 0 || tile > tiles {
				return nil, fmt.Errorf("bridge %d lands on tile %d, off the %d-tile map", i, tile, tiles)
			}
			sizes[tile+1]++
		}
		if pair[0] == pair[1] {
			return nil, fmt.Errorf("bridge %d has both ends on tile %d", i, pair[0])
		}
	}

	starts := make([]uint32, tiles+2)
	for tile := uint32(1); tile < tiles+2; tile++ {
		starts[tile] = starts[tile-1] + sizes[tile]
	}

	ends := make([]uint32, starts[tiles+1])
	next := make([]uint32, tiles+1)
	copy(next, starts)
	for _, pair := range pairs {
		ends[next[pair[0]]] = pair[1]
		next[pair[0]]++
		ends[next[pair[1]]] = pair[0]
		next[pair[1]]++
	}

	return &Bridges{starts: starts, ends: ends}, nil
}

// Across is the far end of each bridge that lands on the tile, a window into the bridges' own table: read it, never
// write it.
func (b *Bridges) Across(tile uint32) []uint32 {
	if int(tile)+1 >= len(b.starts) {
		return nil
	}

	return b.ends[b.starts[tile]:b.starts[tile+1]]
}

func (b *Bridges) Count() int {
	return len(b.ends) / 2
}
