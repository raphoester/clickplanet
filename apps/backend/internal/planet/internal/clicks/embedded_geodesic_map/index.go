package embedded_geodesic_map

import (
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const (
	// f32 rounding tolerance, not a search radius: the closest tiles are 3.09e-3 apart.
	matchEpsilon   = 1e-5
	matchEpsilonSq = matchEpsilon * matchEpsilon

	cellSize = 1e-3
)

type cell struct{ x, y, z int32 }

type positionIndex struct {
	positions []float32
	cells     map[cell][]uint32
}

func newPositionIndex(positions []float32) *positionIndex {
	tiles := len(positions) / 3
	index := &positionIndex{
		positions: positions,
		cells:     make(map[cell][]uint32, tiles),
	}

	for i := range tiles {
		// Wire tile ids are 1-based; the blob is 0-indexed.
		id := uint32(i) + 1

		x := float64(positions[i*3])
		y := float64(positions[i*3+1])
		z := float64(positions[i*3+2])

		lowX, highX := spannedCells(x)
		lowY, highY := spannedCells(y)
		lowZ, highZ := spannedCells(z)

		for cx := lowX; cx <= highX; cx++ {
			for cy := lowY; cy <= highY; cy++ {
				for cz := lowZ; cz <= highZ; cz++ {
					c := cell{cx, cy, cz}
					index.cells[c] = append(index.cells[c], id)
				}
			}
		}
	}

	return index
}

func (index *positionIndex) lookup(v clicks.Vec3) (uint32, bool) {
	c := cell{cellOf(v.X), cellOf(v.Y), cellOf(v.Z)}

	for _, id := range index.cells[c] {
		i := int(id-1) * 3
		dx := float64(index.positions[i]) - v.X
		dy := float64(index.positions[i+1]) - v.Y
		dz := float64(index.positions[i+2]) - v.Z

		if dx*dx+dy*dy+dz*dz <= matchEpsilonSq {
			return id, true
		}
	}

	return 0, false
}

func cellOf(value float64) int32 {
	return int32(math.Floor(value / cellSize))
}

func spannedCells(value float64) (int32, int32) {
	return cellOf(value - matchEpsilon), cellOf(value + matchEpsilon)
}
