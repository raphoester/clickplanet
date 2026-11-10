package clicks

import (
	"fmt"
	"math"
)

// BodyID is one body of water the lattice draws: the ocean is one, the Black Sea another, each lake its own.
type BodyID uint16

// Waters is which bodies of water each coast tile touches, and each body's coast. A tile between two waters is on the
// coast of both.
type Waters struct {
	coasts [][]uint32

	starts []uint32
	bodies []BodyID
}

// NewWaters takes each body's coast tiles, by body.
func NewWaters(tiles uint32, coasts [][]uint32) (*Waters, error) {
	if len(coasts) > math.MaxUint16 {
		return nil, fmt.Errorf("%d bodies of water, past the %d a body id names", len(coasts), math.MaxUint16)
	}

	sizes := make([]uint32, tiles+2)
	for body, coast := range coasts {
		for i, tile := range coast {
			if tile == 0 || tile > tiles {
				return nil, fmt.Errorf("body %d has tile %d on its coast, off the %d-tile map", body, tile, tiles)
			}
			if i > 0 && coast[i-1] >= tile {
				return nil, fmt.Errorf("body %d lists its coast out of order at tile %d", body, tile)
			}
			sizes[tile+1]++
		}
	}

	starts := make([]uint32, tiles+2)
	for tile := uint32(1); tile < tiles+2; tile++ {
		starts[tile] = starts[tile-1] + sizes[tile]
	}

	bodies := make([]BodyID, starts[tiles+1])
	next := make([]uint32, tiles+1)
	copy(next, starts)
	for body, coast := range coasts {
		for _, tile := range coast {
			bodies[next[tile]] = BodyID(body) //nolint:gosec // bounded by math.MaxUint16 above.
			next[tile]++
		}
	}

	return &Waters{coasts: coasts, starts: starts, bodies: bodies}, nil
}

// BodiesOf is a window into the waters' own table: read it, never write it. A tile inland is on no body's coast.
func (w *Waters) BodiesOf(tile uint32) []BodyID {
	if int(tile)+1 >= len(w.starts) {
		return nil
	}

	return w.bodies[w.starts[tile]:w.starts[tile+1]]
}

// CoastOf is a window into the waters' own table: read it, never write it.
func (w *Waters) CoastOf(body BodyID) []uint32 {
	if int(body) >= len(w.coasts) {
		return nil
	}

	return w.coasts[body]
}

func (w *Waters) Bodies() int {
	return len(w.coasts)
}
