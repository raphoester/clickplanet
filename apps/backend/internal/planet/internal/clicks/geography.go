package clicks

import (
	"fmt"
	"slices"
)

// Geography is the map's shape: where every tile is, and which tiles touch it. Read-only once
// built and safe for concurrent use. Where the adjacency was derived from is the adapter's
// business — see clicks/adapters/secondary/geodesic_map.
type Geography struct {
	// Three floats per tile, indexed by (id-1)*3. Kept for later distance and lat/lon bonuses.
	positions []float32

	// CSR: tile id's neighbours are neighbours[starts[id-1]:starts[id]], ascending.
	starts     []uint32
	neighbours []uint32

	stats GeographyStats
}

type GeographyStats struct {
	Tiles uint32

	// Undirected: an edge between two tiles counts once.
	Edges uint32

	// Degrees[n] is how many tiles have exactly n neighbours.
	Degrees [MaxDegree + 1]uint32
}

// MaxDegree is the most neighbours a tile can have. The map is a geodesic honeycomb, so a tile
// has 6 inland and 5 at the 12 icosahedron corners; a coastline takes tiles below that, never
// above it.
const MaxDegree = 6

// Vec3 is a point on the unit sphere.
type Vec3 struct{ X, Y, Z float64 }

// Edge is one tile touching another, in one direction.
type Edge struct{ From, To uint32 }

// NewGeography takes the adjacency an adapter found and builds the map from it. Edges may arrive
// in any order and may repeat — a tile on a seam between two of the lattice's faces is found from
// both — and are expected in both directions.
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

// packEdges checks every edge and returns them as from<<32|to, sorted and deduplicated. One sort
// over that packed form orders by both fields at once, which is the CSR layout already: each
// tile's neighbours come out grouped and ascending.
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

// compressRows lays the sorted edges out as compressed sparse rows: one run per tile, and an index
// of where each run begins, rather than 257,948 slice headers.
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

// countDegrees is the bound check as well as the tally: a seventh neighbour cannot happen on a
// honeycomb, so finding one means the adjacency was not read off one.
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

// checkSymmetry refuses a tile that spreads onto a neighbour which would not spread back.
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

// Neighbours returns the tiles touching id, ascending — and nothing for a lone island or for an id
// outside 1..Tiles, so a caller must have an answer for the empty case. The slice aliases the
// map's own table: read it, never write it.
func (g *Geography) Neighbours(id uint32) []uint32 {
	if id == 0 || id > g.stats.Tiles {
		return nil
	}
	return g.neighbours[g.starts[id-1]:g.starts[id]]
}
