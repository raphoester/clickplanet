package embedded_board

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
)

var errNoBoard = errors.New("this process carries no board")

// Registry is every board this process carries (cpboards.Files): each one's manifest read and checked at boot (Load),
// and its ground (geography, borders, bridges and capitals) built the first time something needs it, about 57 bytes a
// tile.
type Registry struct {
	files fs.FS
	// The carry-overs between two boards (mapdata.Carries).
	carries fs.FS
	boards  []*heldBoard
	// Each carry-over read, by its two boards.
	carried    sync.Mutex
	carryovers map[[2]clicks.BoardName]clicks.Carryover
	flags      func(country string) bool
	logger     *slog.Logger
}

type heldBoard struct {
	board  cpboards.Board
	once   sync.Once
	ground *clicks.Ground
	err    error
}

// New is a registry of the boards in files, read by Load, and of the carry-overs between them in carries. flags is
// whether a player can play for a country: the capitals of the others are dropped.
func New(files fs.FS, carries fs.FS, flags func(country string) bool, logger *slog.Logger) *Registry {
	return &Registry{files: files, carries: carries, flags: flags, logger: logger}
}

// Load reads every board's manifest, refusing no board, and a board whose manifest names a lattice this build cannot
// walk: a tile order it does not know, a warp or corners that are not numbers.
func (r *Registry) Load() error {
	boards, err := cpboards.Boards(r.files)
	if err != nil {
		return fmt.Errorf("failed to read the boards carried: %w", err)
	}
	if len(boards) == 0 {
		return errNoBoard
	}
	held := make([]*heldBoard, 0, len(boards))
	for _, board := range boards {
		if err := checkLattice(board.Manifest); err != nil {
			return fmt.Errorf("%s/%s: %w", board.Manifest.Name, board.Asset, err)
		}
		held = append(held, &heldBoard{board: board})
	}
	r.boards = held
	return nil
}

func checkLattice(m cpboards.Manifest) error {
	_, err := shapeOf(m)
	return err
}

// shapeOf is the lattice the manifest describes: its cols, its warp and rotated corners (three's when it has none) and
// its tile order.
func shapeOf(m cpboards.Manifest) (latticeShape, error) {
	lattice := m.Lattice
	if lattice.Cols < 2 {
		return latticeShape{}, fmt.Errorf("a lattice of %d cols", lattice.Cols)
	}
	if lattice.Order != threesOrder && lattice.Order != canonicalOrder {
		return latticeShape{}, fmt.Errorf("tile order %q is no lattice's", lattice.Order)
	}
	shape := latticeShape{
		cols:    lattice.Cols,
		corners: icosahedronCorners,
		threes:  lattice.Warp == nil && lattice.Corners == nil,
		order:   lattice.Order,
	}
	if warp := lattice.Warp; warp != nil {
		if !finite(warp.A, warp.B, warp.C) {
			return latticeShape{}, fmt.Errorf("a warp of %v, %v, %v", warp.A, warp.B, warp.C)
		}
		shape.warp = &latticeWarp{a: warp.A, b: warp.B, c: warp.C}
	}
	if lattice.Corners != nil {
		if len(lattice.Corners) != cornerCount*3 || !finite(lattice.Corners...) {
			return latticeShape{}, fmt.Errorf("corners of %d numbers, not %d finite ones", len(lattice.Corners), cornerCount*3)
		}
		for corner := range cornerCount {
			at := lattice.Corners[corner*3 : corner*3+3]
			shape.corners[corner] = clicks.Vec3{X: at[0], Y: at[1], Z: at[2]}
		}
	}
	return shape, nil
}

func finite(numbers ...float64) bool {
	for _, number := range numbers {
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return false
		}
	}
	return true
}

// Carried is every board this process carries, as its files name it: its directory, and the hash of its positions in
// tile order (the manifest's positionsHash, which Ground checks the rebuilt positions against). Planet's row of each
// names the same hash, or the boot refuses it.
func (r *Registry) Carried() []eras.Carried {
	carried := make([]eras.Carried, len(r.boards))
	for i, held := range r.boards {
		carried[i] = eras.Carried{Name: clicks.BoardName(held.board.Manifest.Name), Hash: held.board.Hash()}
	}
	return carried
}

// Ground is the ground of the board of that name, built the first time it is asked for and kept: refused for a board
// this process does not carry, and for files that do not rebuild the manifest's tiles.
func (r *Registry) Ground(name clicks.BoardName) (*clicks.Ground, error) {
	for _, held := range r.boards {
		if held.board.Manifest.Name == string(name) {
			held.once.Do(func() { held.ground, held.err = r.load(held.board) })
			return held.ground, held.err
		}
	}
	return nil, fmt.Errorf("%w: %s", eras.ErrNotCarried, name)
}

func (r *Registry) load(board cpboards.Board) (*clicks.Ground, error) {
	started := time.Now()
	name := board.Manifest.Name

	geography, err := loadGeography(board)
	if err != nil {
		return nil, fmt.Errorf("failed to load the geography of board %s: %w", name, err)
	}
	borders, err := loadBorders(board)
	if err != nil {
		return nil, fmt.Errorf("failed to load the borders of board %s: %w", name, err)
	}
	bridges, err := loadBridges(board)
	if err != nil {
		return nil, fmt.Errorf("failed to load the bridges of board %s: %w", name, err)
	}
	capitals, err := loadCapitals(board, borders)
	if err != nil {
		return nil, fmt.Errorf("failed to load the capitals of board %s: %w", name, err)
	}
	capitals = capitals.Flags(r.flags)
	ground, err := clicks.NewGround(clicks.BoardName(name), geography, borders, bridges, capitals)
	if err != nil {
		return nil, fmt.Errorf("failed to build the ground of board %s: %w", name, err)
	}

	stats := geography.Stats()
	r.logger.Info("board loaded",
		slog.String("board", name),
		slog.String("asset", board.Asset),
		slog.Int("tiles", int(stats.Tiles)),
		slog.Int("edges", int(stats.Edges)),
		slog.Any("degrees", stats.Degrees),
		slog.String("borders", board.Manifest.Files.Borders),
		slog.Int("bridges", bridges.Count()),
		slog.Int("capitals", len(capitals.All())),
		slog.Any("took", time.Since(started).Round(time.Millisecond)),
	)

	return ground, nil
}

// A file of the board that holds another tile count than its manifest is another board's.
func checkTiles(board cpboards.Board, asset string, tiles uint32) error {
	if tiles != board.Manifest.Tiles {
		return fmt.Errorf("%s holds %d tiles but %s says %d: a file of another board", asset, tiles, board.Asset,
			board.Manifest.Tiles)
	}
	return nil
}

func loadGeography(board cpboards.Board) (*clicks.Geography, error) {
	positions, tileOf, err := tilesOf(board)
	if err != nil {
		return nil, err
	}
	geography, err := clicks.NewGeography(positions, latticeEdges(board.Manifest.Lattice.Cols, tileOf))
	if err != nil {
		return nil, fmt.Errorf("failed to build the geography: %w", err)
	}
	return geography, nil
}

// tilesOf rebuilds the board's positions from its runs, in tile order, and each lattice point's tile: 0 at sea.
func tilesOf(board cpboards.Board) ([]float32, []uint32, error) {
	manifest := board.Manifest
	blob, err := board.File(manifest.Files.Runs)
	if err != nil {
		return nil, nil, err //nolint:wrapcheck // the board names the file.
	}
	shape, err := shapeOf(manifest)
	if err != nil {
		return nil, nil, err
	}
	cols := shape.cols
	runs, err := decodeRuns(blob, latticeSize(cols))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode %s: %w", manifest.Files.Runs, err)
	}

	isTile := make([]bool, latticeSize(cols))
	for _, run := range runs {
		for id := run.first; id <= run.last; id++ {
			isTile[id] = true
		}
	}

	tileOf := make([]uint32, len(isTile))
	positions := make([]float32, 0, int(manifest.Tiles)*3)
	tile := uint32(0)
	// A tile's id is its rank among the tiles in the board's order, plus one: wire tile ids are 1-based.
	shape.walk(func(face, i, j int, id canonicalID) {
		if !isTile[id] {
			return
		}
		tile++
		tileOf[id] = tile
		position := shape.position(face, i, j)
		positions = append(positions, position[0], position[1], position[2])
	})
	if err := checkTiles(board, manifest.Files.Runs, tile); err != nil {
		return nil, nil, err
	}

	if hash := positionsHash(positions); hash != manifest.PositionsHash {
		return nil, nil, fmt.Errorf("%s rebuilds positions that hash to %08x, %s says %08x: they are not the page's",
			manifest.Files.Runs, hash, board.Asset, manifest.PositionsHash)
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
