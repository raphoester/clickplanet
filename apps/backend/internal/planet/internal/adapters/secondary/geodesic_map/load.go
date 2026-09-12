// Package geodesic_map builds the map's geography from the shipped tile coordinates blob.
//
// Everything in here exists because the adjacency is not shipped — the positions are, and the
// adjacency has to be recovered from them. Ship a precomputed edge list one day and this whole
// package goes while clicks.Geography stays exactly as it is.
//
// The method, the numbers it has to reproduce, and why a radius search is the wrong way to do
// this are in apps/backend/CLAUDE.md under "Map geography".
package geodesic_map

import (
	"fmt"

	mapdata "github.com/raphoester/clickplanet.lol-backend/generated/map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Load builds the geography from the embedded blob and names the blob it used, refusing to start
// rather than serving a map that disagrees with the one players are looking at. expectTiles is
// gameMap.maxIndex.
func Load(expectTiles uint32) (*clicks.Geography, string, error) {
	blob, asset, err := mapdata.Coordinates()
	if err != nil {
		return nil, "", fmt.Errorf("failed to read the embedded coordinates blob: %w", err)
	}

	positions, err := decodeCoordinates(blob)
	if err != nil {
		return nil, asset, fmt.Errorf("failed to decode %s: %w", asset, err)
	}

	tiles := uint32(len(positions) / 3)
	if tiles != expectTiles {
		return nil, asset, fmt.Errorf(
			"%s holds %d tiles but gameMap.maxIndex is %d: the blob and the config were updated apart",
			asset, tiles, expectTiles)
	}

	edges, err := walkLattice(positions)
	if err != nil {
		return nil, asset, fmt.Errorf("failed to read the lattice out of %s: %w", asset, err)
	}

	geography, err := clicks.NewGeography(positions, edges)
	if err != nil {
		return nil, asset, fmt.Errorf("failed to build the geography from %s: %w", asset, err)
	}

	return geography, asset, nil
}

// walkLattice resolves every vertex of all 20 icosahedron faces back to a tile and reports the
// six ±1 exchanges between barycentric coordinates as its neighbours.
func walkLattice(positions []float32) ([]clicks.Edge, error) {
	tiles := uint32(len(positions) / 3)
	index := newPositionIndex(positions)

	resolved := make([]bool, tiles)
	edges := make([]clicks.Edge, 0, 2<<20)

	// One face's lattice, (i, j) -> tile id, 0 over water. Reused across the 20 faces.
	face := make([]uint32, faceVertices)

	for _, corners := range icosahedronFaces {
		a := icosahedronVertices[corners[0]]
		b := icosahedronVertices[corners[1]]
		c := icosahedronVertices[corners[2]]

		for i := 0; i <= Cols; i++ {
			for j := 0; j <= Cols-i; j++ {
				// i is r and j is q; p is whatever is left of Cols.
				position := latticePosition(a, b, c, Cols-i-j, j, i)

				id, ok := index.lookup(position)
				face[faceIndex(i, j)] = id
				if ok {
					resolved[id-1] = true
				}
			}
		}

		for i := 0; i <= Cols; i++ {
			for j := 0; j <= Cols-i; j++ {
				from := face[faceIndex(i, j)]
				if from == 0 {
					continue
				}

				for _, move := range latticeMoves {
					ni, nj := i+move[0], j+move[1]
					// Off this face; another face in the walk carries that vertex.
					if ni < 0 || nj < 0 || ni+nj > Cols {
						continue
					}

					to := face[faceIndex(ni, nj)]
					if to == 0 {
						continue
					}

					edges = append(edges, clicks.Edge{From: from, To: to})
				}
			}
		}
	}

	// A tile on no lattice vertex means the blob was generated at a different subdivision.
	if missing := countFalse(resolved); missing > 0 {
		return nil, fmt.Errorf(
			"%d of %d tiles do not sit on the detail-%d lattice: the blob was generated at a different subdivision",
			missing, tiles, Detail)
	}

	return edges, nil
}

func countFalse(flags []bool) int {
	count := 0
	for _, flag := range flags {
		if !flag {
			count++
		}
	}
	return count
}
