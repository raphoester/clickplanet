package clicks

import (
	"math"
	"slices"
)

// One cell per degree: row 0 is the north pole, column 0 is 180° west.
const (
	GridColumns = 360
	GridRows    = 180
)

// A disc a little wider than half the spacing covers the gap between two tiles.
const gridReach = 0.6

// A tile is drawn in every cell its disc covers, so no cell of land far north is left empty.
type Grid struct {
	starts []uint32
	cells  []uint32
	land   []byte
}

func NewGrid(geography *Geography, spacing float64) *Grid {
	tiles := geography.Stats().Tiles
	grid := &Grid{starts: make([]uint32, tiles+1), land: make([]byte, GridColumns*GridRows)}
	radius := gridReach * spacing * 180 / math.Pi

	for id := uint32(1); id <= tiles; id++ {
		position, _ := geography.Position(id)
		grid.cells = appendCells(grid.cells, position, radius)
		grid.starts[id] = uint32(len(grid.cells)) //nolint:gosec // a few cells per tile of a uint32 map.
	}

	for _, cell := range grid.cells {
		if grid.land[cell] < math.MaxUint8 {
			grid.land[cell]++
		}
	}

	return grid
}

func appendCells(cells []uint32, position Vec3, radius float64) []uint32 {
	unit := position.Unit()
	latitude := math.Asin(min(max(unit.Y, -1), 1)) * 180 / math.Pi
	longitude := math.Atan2(unit.Z, -unit.X) * 180 / math.Pi

	first, last := 0, GridColumns-1
	if span := radius / max(math.Cos(latitude*math.Pi/180), 1e-9); span < 180 {
		first = int(math.Floor(longitude - span + 180))
		last = int(math.Floor(longitude + span + 180))
	}
	if last-first >= GridColumns {
		first, last = 0, GridColumns-1
	}

	for row := rowOf(latitude + radius); row <= rowOf(latitude-radius); row++ {
		for column := first; column <= last; column++ {
			cells = append(cells, uint32(row*GridColumns+(column%GridColumns+GridColumns)%GridColumns)) //nolint:gosec // a cell of the grid.
		}
	}

	return cells
}

func rowOf(latitude float64) int {
	return min(max(int(math.Floor(90-latitude)), 0), GridRows-1)
}

// At most 255 tiles a cell.
func (g *Grid) Land() []byte {
	return slices.Clone(g.land)
}

func (g *Grid) Heat(changes map[uint32]int) []uint32 {
	heat := make([]uint32, GridColumns*GridRows)

	for tile, times := range changes {
		if tile == 0 || int(tile) >= len(g.starts) {
			continue
		}
		for _, cell := range g.cells[g.starts[tile-1]:g.starts[tile]] {
			heat[cell] += uint32(times) //nolint:gosec // a count of changes, never negative.
		}
	}

	return heat
}
