package embedded_board

import (
	"encoding/json"
	"fmt"

	mapdata "github.com/raphoester/clickplanet.lol-backend/generated/map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// The manifest of a board (/map/README.md).
type manifest struct {
	Name    string `json:"name"`
	Lattice struct {
		Cols    int       `json:"cols"`
		Warp    *struct{} `json:"warp"`
		Corners []float64 `json:"corners"`
		Order   string    `json:"order"`
	} `json:"lattice"`
	Tiles         uint32 `json:"tiles"`
	PositionsHash uint32 `json:"positionsHash"`
	Files         struct {
		Runs     string `json:"runs"`
		Borders  string `json:"borders"`
		Capitals string `json:"capitals"`
		Bridges  string `json:"bridges"`
	} `json:"files"`
}

type board struct {
	name     string
	asset    string
	manifest manifest
}

// board reads the board this app plays, refusing one of another size than gameMap.maxIndex.
func (l *Loader) board() (board, error) {
	names, err := mapdata.Boards()
	if err != nil {
		return board{}, fmt.Errorf("failed to list the embedded boards: %w", err)
	}
	if len(names) != 1 {
		return board{}, fmt.Errorf("expected exactly one embedded board, found %d: %v", len(names), names)
	}

	blob, asset, err := mapdata.Manifest(names[0])
	if err != nil {
		return board{}, fmt.Errorf("failed to read board %s's manifest: %w", names[0], err)
	}

	var read manifest
	if err := json.Unmarshal(blob, &read); err != nil {
		return board{}, fmt.Errorf("failed to parse %s: %w", asset, err)
	}
	if read.Name != names[0] {
		return board{}, fmt.Errorf("%s names board %q, in the directory of %q", asset, read.Name, names[0])
	}
	if read.Tiles != l.expectTiles {
		return board{}, fmt.Errorf(
			"%s holds %d tiles but gameMap.maxIndex is %d: the board and the config were updated apart",
			asset, read.Tiles, l.expectTiles)
	}
	if err := read.checkLattice(); err != nil {
		return board{}, fmt.Errorf("%s: %w", asset, err)
	}

	return board{name: names[0], asset: asset, manifest: read}, nil
}

func (m manifest) checkLattice() error {
	if m.Lattice.Cols < 2 {
		return fmt.Errorf("a lattice of %d cols", m.Lattice.Cols)
	}
	if m.Lattice.Warp != nil || m.Lattice.Corners != nil {
		return fmt.Errorf("a warped or rotated lattice is not built yet")
	}
	if m.Lattice.Order != "three" {
		return fmt.Errorf("tile order %q is not built yet", m.Lattice.Order)
	}
	return nil
}

func (b board) file(name string) ([]byte, error) {
	blob, err := mapdata.File(b.name, name)
	if err != nil {
		return nil, fmt.Errorf("failed to read a file of %s: %w", b.asset, err)
	}
	return blob, nil
}

// tiles rebuilds the board's positions from its runs, in tile order, and each lattice point's tile: 0 at sea.
func (b board) tiles() ([]float32, []uint32, error) {
	blob, err := b.file(b.manifest.Files.Runs)
	if err != nil {
		return nil, nil, err
	}
	cols := b.manifest.Lattice.Cols
	runs, err := decodeRuns(blob, latticeSize(cols))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode %s: %w", b.manifest.Files.Runs, err)
	}

	isTile := make([]bool, latticeSize(cols))
	for _, run := range runs {
		for id := run.first; id <= run.last; id++ {
			isTile[id] = true
		}
	}

	tileOf := make([]uint32, len(isTile))
	positions := make([]float32, 0, int(b.manifest.Tiles)*3)
	tile := uint32(0)
	walkLattice(cols, func(face, i, j int, id canonicalID) {
		if !isTile[id] {
			return
		}
		// Wire tile ids are 1-based.
		tile++
		tileOf[id] = tile
		position := latticePosition(cols, face, i, j)
		positions = append(positions, position[0], position[1], position[2])
	})
	if tile != b.manifest.Tiles {
		return nil, nil, fmt.Errorf("%s holds %d tiles, %s says %d", b.manifest.Files.Runs, tile, b.asset, b.manifest.Tiles)
	}

	if hash := positionsHash(positions); hash != b.manifest.PositionsHash {
		return nil, nil, fmt.Errorf("%s rebuilds positions that hash to %08x, %s says %08x: they are not the page's",
			b.manifest.Files.Runs, hash, b.asset, b.manifest.PositionsHash)
	}

	return positions, tileOf, nil
}

// latticeEdges are the ±1 exchanges of (face, i, j) between two tiles, each found from both ends.
func latticeEdges(cols int, tileOf []uint32) []clicks.Edge {
	ids := latticeFaceIDs(cols)
	edges := make([]clicks.Edge, 0, 2<<20)

	for face := range faceCount {
		for i := 0; i <= cols; i++ {
			for j := 0; j <= cols-i; j++ {
				from := tileOf[ids[face].of(cols, i, j)]
				if from == 0 {
					continue
				}

				for _, move := range latticeMoves {
					ni, nj := i+move[0], j+move[1]
					if ni < 0 || nj < 0 || ni+nj > cols {
						continue
					}
					if to := tileOf[ids[face].of(cols, ni, nj)]; to != 0 {
						edges = append(edges, clicks.Edge{From: from, To: to})
					}
				}
			}
		}
	}

	return edges
}
