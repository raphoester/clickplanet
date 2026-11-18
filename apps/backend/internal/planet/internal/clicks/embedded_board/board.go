package embedded_board

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
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
	files  fs.FS
	boards []*heldBoard
	flags  func(country string) bool
	logger *slog.Logger
}

type heldBoard struct {
	board  cpboards.Board
	once   sync.Once
	ground *clicks.Ground
	err    error
}

// New is a registry of the boards in files, read by Load. flags is whether a player can play for a country: the capitals
// of the others are dropped.
func New(files fs.FS, flags func(country string) bool, logger *slog.Logger) *Registry {
	return &Registry{files: files, flags: flags, logger: logger}
}

// Load reads every board's manifest, refusing no board, and a board whose manifest names a lattice this build cannot
// walk.
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
	cols := manifest.Lattice.Cols
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
