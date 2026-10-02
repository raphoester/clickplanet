package clicks

import (
	"fmt"
	"slices"
)

type Geography struct {
	positions []float32

	starts     []uint32
	neighbours []uint32

	stats GeographyStats
}

type GeographyStats struct {
	Tiles uint32

	Edges uint32

	Degrees [MaxDegree + 1]uint32
}

const MaxDegree = 6

type Vec3 struct{ X, Y, Z float64 }

type Edge struct{ From, To uint32 }

func NewGeography(positions []float32, edges []Edge) (*Geography, error) {
	tiles := uint32(len(positions) / 3)
	if tiles == 0 {
		return nil, fmt.Errorf("a map with no tiles")
	}

	adjacency, err := packEdges(edges, tiles)
	if err != nil {
		return nil, err
	}

	starts, neighbours := compressRows(adjacency, tiles)
	geography := &Geography{
		positions:  positions,
		starts:     starts,
		neighbours: neighbours,
		stats:      GeographyStats{Tiles: tiles, Edges: uint32(len(adjacency) / 2)},
	}

	degrees, err := geography.countDegrees()
	if err != nil {
		return nil, err
	}
	geography.stats.Degrees = degrees

	if err := geography.checkSymmetry(); err != nil {
		return nil, err
	}

	return geography, nil
}

func packEdges(edges []Edge, tiles uint32) ([]uint64, error) {
	packed := make([]uint64, len(edges))

	for i, edge := range edges {
		if edge.From == 0 || edge.From > tiles || edge.To == 0 || edge.To > tiles {
			return nil, fmt.Errorf("edge %d→%d is outside 1..%d", edge.From, edge.To, tiles)
		}
		if edge.From == edge.To {
			return nil, fmt.Errorf("tile %d touches itself", edge.From)
		}
		packed[i] = uint64(edge.From)<<32 | uint64(edge.To)
	}

	slices.Sort(packed)

	return slices.Compact(packed), nil
}

func compressRows(packed []uint64, tiles uint32) (starts, neighbours []uint32) {
	starts = make([]uint32, tiles+1)
	neighbours = make([]uint32, len(packed))

	for i, edge := range packed {
		neighbours[i] = uint32(edge)
		starts[uint32(edge>>32)]++
	}

	for i := 1; i <= int(tiles); i++ {
		starts[i] += starts[i-1]
	}

	return starts, neighbours
}

func (g *Geography) countDegrees() ([MaxDegree + 1]uint32, error) {
	var degrees [MaxDegree + 1]uint32

	for id := uint32(1); id <= g.stats.Tiles; id++ {
		degree := len(g.Neighbours(id))
		if degree > MaxDegree {
			return degrees, fmt.Errorf("tile %d has %d neighbours, more than the %d a tile can have",
				id, degree, MaxDegree)
		}
		degrees[degree]++
	}

	return degrees, nil
}

func (g *Geography) checkSymmetry() error {
	for id := uint32(1); id <= g.stats.Tiles; id++ {
		for _, other := range g.Neighbours(id) {
			if _, found := slices.BinarySearch(g.Neighbours(other), id); !found {
				return fmt.Errorf("tile %d lists %d as a neighbour but %d does not list %d",
					id, other, other, id)
			}
		}
	}

	return nil
}

func (g *Geography) Stats() GeographyStats {
	return g.stats
}

// The slice aliases the map's own table: never write to it.
func (g *Geography) Neighbours(id uint32) []uint32 {
	if id == 0 || id > g.stats.Tiles {
		return nil
	}
	return g.neighbours[g.starts[id-1]:g.starts[id]]
}
